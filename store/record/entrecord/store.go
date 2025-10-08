package entrecord

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/record/entrecord/ent"
	"github.com/r3dpixel/card-client/store/record/entrecord/ent/recordentity"
	"github.com/r3dpixel/card-client/store/record/entrecord/ent/tagentity"
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
	foreignKeysPragma       = "_fk=1"
	cachedConnectionsPragma = "cache=shared"
)

type Store struct {
	client        *ent.Client
	ctx           context.Context
	isTransaction bool
}

func buildDSN(opts *Options) string {
	dsn := opts.DatabasePath

	pragmas := []string{foreignKeysPragma}
	if opts.CacheConnections {
		pragmas = append(pragmas, cachedConnectionsPragma)
	}

	return fmt.Sprintf("file:%s?%s", dsn, strings.Join(pragmas, "&"))
}

func NewStore(opts Options) (record.Store, error) {

	driver, err := sql.Open(dialect.SQLite, buildDSN(&opts))
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
		client: client,
		ctx:    context.Background(),
	}, nil
}

func (s *Store) Count(filter resource.Filter) int {
	count, _ := fBuilder.ApplyFilter(s.client.RecordEntity.Query(), filter).Count(s.ctx)
	return count
}

func (s *Store) FindPagedRIDs(filter resource.Filter, offset int, limit int) []resource.RID {
	recordIDs, err := fBuilder.ApplyFilter(s.client.RecordEntity.Query(), filter).
		Offset(offset).
		Limit(limit).
		IDs(s.ctx)
	if err != nil {
		return nil
	}
	return recordIDs
}

func (s *Store) FindRecords(rids []resource.RID) resource.Box[resource.Record] {
	readAt := timestamp.Now[timestamp.Nano]()

	entities, _ := s.client.RecordEntity.
		Query().
		WithTags(func(q *ent.TagEntityQuery) {
			q.Order(ent.Asc(tagentity.FieldID))
		}).
		WithCreator().
		Where(recordentity.IDIn(rids...)).
		All(s.ctx)

	records := make([]resource.Record, len(entities))
	for i := range entities {
		mapRecordEntity(entities[i], &records[i])
	}

	return resource.Box[resource.Record]{
		Items:     records,
		Timestamp: readAt,
	}
}

func (s *Store) FindExportHeaders(rids []resource.RID) resource.Box[resource.ExportHeader] {
	readAt := timestamp.Now[timestamp.Nano]()

	entities, _ := s.client.RecordEntity.
		Query().
		Where(recordentity.IDIn(rids...)).
		Select(recordentity.FieldID, recordentity.FieldExportTime, recordentity.FieldExportedVersion).
		All(s.ctx)

	headers := make([]resource.ExportHeader, len(entities))
	for index, entity := range entities {
		headers[index] = resource.ExportHeader{
			ID: entity.ID,
			ExportData: resource.ExportData{
				ExportTime:      entity.ExportTime,
				ExportedVersion: entity.ExportedVersion,
			},
		}
	}

	return resource.Box[resource.ExportHeader]{
		Items:     headers,
		Timestamp: readAt,
	}
}

func (s *Store) FindURLs(normalizedURLs []string) []string {
	existingURLs, _ := s.client.RecordEntity.
		Query().
		Where(recordentity.NormalizedURLIn(normalizedURLs...)).
		Select(recordentity.FieldNormalizedURL).
		Strings(s.ctx)

	return existingURLs
}

func (s *Store) InsertRecord(metadata *models.Metadata, importData resource.ImportData) error {
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

		return s.client.RecordEntity.Create().
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
			Exec(s.ctx)
	})
}

func (s *Store) UpdateRecord(rid resource.RID, metadata *models.Metadata, syncData resource.SyncData) error {
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
			SetSyncStatus(syncData.SyncStatus).
			SetSyncTime(syncData.SyncTime).
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

func (s *Store) UpdateFavoriteData(rids []resource.RID, favorite bool) error {
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

func (s *Store) WithContext(ctx context.Context) record.Store {
	return &Store{
		client: s.client,
		ctx:    ctx,
	}
}

func (s *Store) WithTx(op func(store record.TxStore) error) error {
	if s.isTransaction {
		return op(s)
	}

	tx, err := s.client.Tx(s.ctx)
	if err != nil {
		return err
	}

	txStore := Store{
		client:        tx.Client(),
		ctx:           s.ctx,
		isTransaction: true,
	}

	if err := op(&txStore); err != nil {
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
