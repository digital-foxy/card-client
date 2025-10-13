package erecord

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/record/erecord/ent"
	"github.com/r3dpixel/card-client/store/record/erecord/ent/recordentity"
	"github.com/r3dpixel/card-client/store/record/erecord/ent/tagentity"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/slicesx"
	"github.com/r3dpixel/toolkit/stringsx"
	"github.com/r3dpixel/toolkit/symbols"
	"github.com/r3dpixel/toolkit/timestamp"
	_ "github.com/sqlite3ent/sqlite3"
)

var (
	fBuilder     *filterBuilder
	fBuilderOnce sync.Once
)

const (
	defaultMaxConnections   = 3
	foreignKeysPragma       = "_fk=1"
	cachedConnectionsPragma = "cache=shared"
)

type Builder Options

func (b Builder) Build(path string) (record.Store, error) {
	return New(path, Options(b))
}

type Options struct {
	CacheConnections   bool
	MaxConnections     int
	MaxIdleConnections int
	MaxLifetime        time.Duration
}

type Store struct {
	client        *ent.Client
	ctx           context.Context
	isTransaction bool
}

func InMemoryStore() (record.Store, error) {
	return New(":memory:", Options{
		CacheConnections:   true,
		MaxConnections:     3,
		MaxIdleConnections: 3,
		MaxLifetime:        0,
	})
}

func New(path string, opts Options) (record.Store, error) {
	if opts.MaxConnections <= 0 {
		opts.MaxConnections = defaultMaxConnections
	}

	driver, err := sql.Open(dialect.SQLite, buildDSN(path, &opts))
	if err != nil {
		return nil, err
	}

	db := driver.DB()
	db.SetMaxOpenConns(opts.MaxConnections)
	db.SetMaxIdleConns(opts.MaxIdleConnections)
	db.SetConnMaxLifetime(opts.MaxLifetime)

	client := ent.NewClient(ent.Driver(driver))

	if err = client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		return nil, err
	}

	fBuilderOnce.Do(func() {
		fBuilder = newFilterBuilder()
	})

	return &Store{
		client:        client,
		ctx:           context.Background(),
		isTransaction: false,
	}, nil
}

func buildDSN(path string, opts *Options) string {
	pragmas := []string{foreignKeysPragma}
	if opts.CacheConnections {
		pragmas = append(pragmas, cachedConnectionsPragma)
	}

	return fmt.Sprintf("file:%s?%s", path, strings.Join(pragmas, "&"))
}

func (s *Store) Count(filter resource.Filter) (int, error) {
	return fBuilder.ApplyFilter(s.client.RecordEntity.Query(), filter).Count(s.ctx)
}

func (s *Store) FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error) {
	return fBuilder.ApplyFilter(s.client.RecordEntity.Query(), filter).
		Offset(offset).
		Limit(limit).
		IDs(s.ctx)
}

func (s *Store) FindRecord(rid resource.RID) (*resource.Record, error) {
	entity, err := s.client.RecordEntity.
		Query().
		WithTags(func(q *ent.TagEntityQuery) {
			q.Order(ent.Asc(tagentity.FieldID))
		}).
		WithCreator().
		Where(recordentity.ID(rid)).
		First(s.ctx)
	if err != nil {
		return nil, err
	}

	var rec resource.Record
	mapRecord(entity, &rec)

	return &rec, nil
}

func (s *Store) FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error) {
	readAt := timestamp.Now[timestamp.Nano]()

	entities, err := s.client.RecordEntity.
		Query().
		WithTags(func(q *ent.TagEntityQuery) {
			q.Order(ent.Asc(tagentity.FieldID))
		}).
		WithCreator().
		Where(recordentity.IDIn(rids...)).
		All(s.ctx)
	if err != nil {
		return resource.Box[resource.Record]{}, err
	}

	records := make([]resource.Record, len(entities))
	for index := range entities {
		mapRecord(entities[index], &records[index])
	}

	return resource.Box[resource.Record]{
		Items:     records,
		Timestamp: readAt,
	}, nil
}

func (s *Store) FindExportHeaders(rids ...resource.RID) (resource.Box[resource.ExportHeader], error) {
	readAt := timestamp.Now[timestamp.Nano]()

	entities, err := s.client.RecordEntity.
		Query().
		Where(recordentity.IDIn(rids...)).
		Select(recordentity.FieldID, recordentity.FieldExportTime, recordentity.FieldExportedVersion).
		All(s.ctx)
	if err != nil {
		return resource.Box[resource.ExportHeader]{}, err
	}

	headers := make([]resource.ExportHeader, len(entities))
	for index := range entities {
		mapExportHeader(entities[index], &headers[index])
	}

	return resource.Box[resource.ExportHeader]{
		Items:     headers,
		Timestamp: readAt,
	}, nil
}

func (s *Store) FindURLs(normalizedURLs ...string) ([]string, error) {
	return s.client.RecordEntity.
		Query().
		Where(recordentity.NormalizedURLIn(normalizedURLs...)).
		Select(recordentity.FieldNormalizedURL).
		Strings(s.ctx)
}

func (s *Store) FindTagNames(tids ...resource.TID) ([]string, error) {
	return s.client.TagEntity.
		Query().
		Where(tagentity.IDIn(tids...)).
		Select(tagentity.FieldName).
		Strings(s.ctx)
}

func (s *Store) InsertRecord(metadata *models.Metadata, importData resource.ImportData) (resource.RID, error) {
	var rid resource.RID
	err := s.WithTx(func(store record.TxStore) error {
		if err := s.upsertTags(metadata.Tags); err != nil {
			return err
		}
		tids := slicesx.Map(metadata.Tags, func(tag models.Tag) resource.TID {
			return resource.TID(tag.Slug)
		})

		cid, err := s.upsertCreator(metadata.Source, metadata.CreatorInfo)
		if err != nil {
			return err
		}

		entity, err := s.client.RecordEntity.Create().
			SetImportTime(importData.ImportTime).
			SetImportIndex(importData.ImportIndex).
			SetSource(metadata.Source).
			SetNormalizedURL(metadata.NormalizedURL).
			SetDirectURL(metadata.DirectURL).
			SetPlatformID(metadata.CardInfo.PlatformID).
			SetCharacterID(metadata.CharacterID).
			SetName(metadata.Name).
			SetTitle(metadata.Title).
			SetTagline(metadata.Tagline).
			SetCreateTime(metadata.CreateTime).
			SetUpdateTime(metadata.UpdateTime).
			SetBookUpdateTime(metadata.BookUpdateTime).
			AddTagIDs(tids...).
			SetCreatorID(cid).
			SetSyncStatus(resource.SyncSuccess).
			SetSyncTime(importData.ImportTime).
			SetExportTime(timestamp.Nano(-1)).
			SetExportedVersion(timestamp.Nano(-1)).
			SetFavorite(false).
			Save(s.ctx)

		rid = entity.ID
		return err
	})

	return rid, err
}

func (s *Store) UpdateRecord(rid resource.RID, metadata *models.Metadata, syncTime timestamp.Nano) error {
	return s.WithTx(func(store record.TxStore) error {
		if err := s.upsertTags(metadata.Tags); err != nil {
			return err
		}
		tids := slicesx.Map(metadata.Tags, func(tag models.Tag) resource.TID {
			return resource.TID(tag.Slug)
		})

		cid, err := s.upsertCreator(metadata.Source, metadata.CreatorInfo)
		if err != nil {
			return err
		}

		return s.client.RecordEntity.UpdateOneID(rid).
			SetSource(metadata.Source).
			SetNormalizedURL(metadata.NormalizedURL).
			SetDirectURL(metadata.DirectURL).
			SetPlatformID(metadata.CardInfo.PlatformID).
			SetCharacterID(metadata.CharacterID).
			SetName(metadata.Name).
			SetTitle(metadata.Title).
			SetTagline(metadata.Tagline).
			SetCreateTime(metadata.CreateTime).
			SetUpdateTime(metadata.UpdateTime).
			SetBookUpdateTime(metadata.BookUpdateTime).
			ClearTags().
			AddTagIDs(tids...).
			SetCreatorID(cid).
			SetSyncStatus(resource.SyncSuccess).
			SetSyncTime(syncTime).
			Exec(s.ctx)
	})
}

func (s *Store) UpdateSyncData(rid resource.RID, syncData resource.SyncData) error {
	return s.client.RecordEntity.UpdateOneID(rid).
		SetSyncStatus(syncData.SyncStatus).
		SetSyncTime(syncData.SyncTime).
		Exec(s.ctx)
}

func (s *Store) UpdateExportData(rid resource.RID, exportData resource.ExportData) error {
	return s.client.RecordEntity.UpdateOneID(rid).
		SetExportTime(exportData.ExportTime).
		SetExportedVersion(exportData.ExportedVersion).
		Exec(s.ctx)
}

func (s *Store) UpdateFavoriteData(favorite bool, rids ...resource.RID) error {
	return s.client.RecordEntity.Update().
		Where(recordentity.IDIn(rids...)).
		SetFavorite(favorite).
		Exec(s.ctx)
}

func (s *Store) ToggleFavorite(rid resource.RID) error {
	return s.client.RecordEntity.UpdateOneID(rid).
		Modify(func(u *sql.UpdateBuilder) {
			u.Set(recordentity.FieldFavorite, sql.Expr("NOT "+recordentity.FieldFavorite))
		}).
		Exec(s.ctx)
}

func (s *Store) WithContext(ctx context.Context) record.CtxStore {
	ctxStore := *s
	ctxStore.ctx = ctx
	return &ctxStore
}

func (s *Store) WithTx(op func(store record.TxStore) error) error {
	if s.isTransaction {
		return op(s)
	}

	tx, err := s.client.Tx(s.ctx)
	if err != nil {
		return err
	}

	if err := op(s.getTxStore(tx.Client())); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) upsertTags(tags []models.Tag) error {
	return s.client.TagEntity.
		MapCreateBulk(
			tags,
			func(c *ent.TagEntityCreate, index int) {
				c.SetID(resource.TID(tags[index].Slug)).SetName(tags[index].Name)
			},
		).
		OnConflict().
		DoNothing().
		Exec(s.ctx)
}

func (s *Store) upsertCreator(source source.ID, creatorInfo models.CreatorInfo) (resource.CID, error) {
	cid := resource.CID(string(source) + symbols.Colon + creatorInfo.PlatformID)

	err := s.client.CreatorEntity.Create().
		SetID(cid).
		SetNickname(creatorInfo.Nickname).
		SetUsername(creatorInfo.Username).
		SetPlatformID(creatorInfo.PlatformID).
		SetSource(source).
		OnConflict().
		UpdateNewValues().
		Exec(s.ctx)

	if err != nil {
		return resource.CID(stringsx.Empty), err
	}

	return cid, nil
}

func (s *Store) getTxStore(txClient *ent.Client) *Store {
	txStore := *s
	txStore.client = txClient
	txStore.isTransaction = true
	return &txStore
}
