package erecord

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/mattn/go-sqlite3"
	"github.com/digital-foxy/card-client/store/record"
	"github.com/digital-foxy/card-client/store/record/erecord/ent"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/creatorentity"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/recordentity"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/tagentity"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/slicesx"
	"github.com/digital-foxy/toolkit/stringsx"
	"github.com/digital-foxy/toolkit/symbols"
	"github.com/digital-foxy/toolkit/timestamp"
)

type void = struct{}

var (
	fBuilder         *filterBuilder
	fBuilderOnce     sync.Once
	sqlContentFields = strings.Join(slicesx.PrependValue("id", resource.ContentFields), ", ")
)

const (
	defaultMaxConnections   = 3
	foreignKeysPragma       = "_fk=1"
	cachedConnectionsPragma = "cache=shared"
	walModePragma           = "_journal_mode=WAL"
	sqliteDriverName        = "sqlite3_with_regexp"
)

func init() {
	// Register custom SQLite driver with REGEXP support
	sql.Register(sqliteDriverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("regexp", func(pattern, text string) (bool, error) {
				return regexp.MatchString(pattern, text)
			}, true)
		},
	})
}

// Builder implements record.Builder using Ent/SQLite
type Builder Options

func (b Builder) Build(path string) (record.Store, error) {
	return New(path, Options(b))
}

// Options configures the SQLite record store
type Options struct {
	CacheConnections   bool
	MaxConnections     int
	MaxIdleConnections int
	MaxLifetime        time.Duration
}

// Store implements record.Store using Ent ORM with SQLite
type Store struct {
	client        *ent.Client
	ctx           context.Context
	isTransaction bool
	mu            *sync.RWMutex // Shared across tx copies to coordinate SQLite access
}

// InMemoryStore creates an in-memory SQLite store for testing
func InMemoryStore() (record.Store, error) {
	return New(":memory:", Options{
		CacheConnections:   true,
		MaxConnections:     3,
		MaxIdleConnections: 3,
		MaxLifetime:        0,
	})
}

// New creates a new SQLite record store at the given path
func New(path string, opts Options) (record.Store, error) {
	if opts.MaxConnections <= 0 {
		opts.MaxConnections = defaultMaxConnections
	}

	driver, err := entsql.Open(sqliteDriverName, buildDSN(path, &opts))
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

	if _, err = db.Exec(fmt.Sprintf(`
          CREATE VIRTUAL TABLE IF NOT EXISTS fts USING fts5(
              %s,
              tokenize="unicode61 remove_diacritics 2"
          );

          CREATE TRIGGER IF NOT EXISTS fts_tr AFTER DELETE ON records BEGIN
              DELETE FROM fts WHERE id = old.id;
          END;
          `, sqlContentFields),
	); err != nil {
		return nil, err
	}

	return &Store{
		client:        client,
		ctx:           context.Background(),
		isTransaction: false,
		mu:            &sync.RWMutex{},
	}, nil
}

func buildDSN(path string, opts *Options) string {
	pragmas := []string{foreignKeysPragma, walModePragma}
	if opts.CacheConnections {
		pragmas = append(pragmas, cachedConnectionsPragma)
	}

	return fmt.Sprintf("file:%s?%s", path, strings.Join(pragmas, "&"))
}

func (s *Store) Count(filter resource.Filter) (int, error) {
	return withLock(s, false, func() (int, error) {
		return fBuilder.ApplyFilter(s.client.RecordEntity.Query(), filter).Count(s.ctx)
	})
}

func (s *Store) FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error) {
	return withLock(s, false, func() ([]resource.RID, error) {
		return fBuilder.ApplyFilter(s.client.RecordEntity.Query(), filter).
			Offset(offset).
			Limit(limit).
			IDs(s.ctx)
	})
}

func (s *Store) FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error) {
	return withLock(s, false, func() (resource.Box[resource.Record], error) {
		readAt := timestamp.NowNano()

		entities, err := s.client.RecordEntity.
			Query().
			WithTags(func(q *ent.TagEntityQuery) {
				q.Order(ent.Asc(tagentity.FieldID))
			}).
			WithCreator().
			Where(recordentity.IDIn(rids...)).
			All(s.ctx)
		if err != nil {
			return resource.Box[resource.Record]{
				Timestamp: readAt,
			}, err
		}

		records := make([]resource.Record, len(entities))
		for index := range entities {
			mapRecord(entities[index], &records[index])
		}

		return resource.Box[resource.Record]{
			Items:     records,
			Timestamp: readAt,
		}, nil
	})
}

func (s *Store) FindRecord(rid resource.RID) (*resource.Record, error) {
	return withLock(s, false, func() (*resource.Record, error) {
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
	})
}

func (s *Store) FindCreatorByCID(cid resource.CID) (resource.Creator, error) {
	return withLock(s, false, func() (resource.Creator, error) {
		entity, err := s.client.CreatorEntity.
			Query().
			Where(creatorentity.ID(cid)).
			First(s.ctx)
		if err != nil {
			return resource.Creator{}, err
		}

		return transformCreator(entity), nil
	})
}

func (s *Store) FindCreatorByNickname(source source.ID, nickname string) (resource.Creator, error) {
	return withLock(s, false, func() (resource.Creator, error) {
		entity, err := s.client.CreatorEntity.
			Query().
			Where(creatorentity.SourceEQ(source), creatorentity.NicknameEQ(nickname)).
			First(s.ctx)
		if err != nil {
			return resource.Creator{}, err
		}

		return transformCreator(entity), nil
	})
}

func (s *Store) FindURLs(normalizedURLs ...string) ([]string, error) {
	return withLock(s, false, func() ([]string, error) {
		return s.client.RecordEntity.
			Query().
			Where(recordentity.NormalizedURLIn(normalizedURLs...)).
			Select(recordentity.FieldNormalizedURL).
			Strings(s.ctx)
	})
}

func (s *Store) FindTagNames(tids ...resource.TID) ([]string, error) {
	return withLock(s, false, func() ([]string, error) {
		return s.client.TagEntity.
			Query().
			Where(tagentity.IDIn(tids...)).
			Select(tagentity.FieldName).
			Strings(s.ctx)
	})
}

func (s *Store) TagNames() ([]string, error) {
	return withLock(s, false, func() ([]string, error) {
		return s.client.TagEntity.
			Query().
			Select(tagentity.FieldName).
			Strings(s.ctx)
	})
}

func (s *Store) CID(source source.ID, platformID string) resource.CID {
	return resource.CID(string(source) + symbols.Colon + platformID)
}

func (s *Store) UpsertTags(tags []models.Tag) error {
	_, err := withLock(s, true, func() (void, error) {
		return void{}, s.client.TagEntity.
			MapCreateBulk(
				tags,
				func(c *ent.TagEntityCreate, index int) {
					c.SetID(resource.TID(tags[index].Slug)).SetName(tags[index].Name)
				},
			).
			OnConflict().
			DoNothing().
			Exec(s.ctx)
	})
	return err
}

func (s *Store) UpsertCreator(source source.ID, creatorInfo models.CreatorInfo) (resource.CID, error) {
	cid := s.CID(source, creatorInfo.PlatformID)

	_, err := withLock(s, true, func() (void, error) {
		return void{}, s.client.CreatorEntity.Create().
			SetID(cid).
			SetNickname(creatorInfo.Nickname).
			SetUsername(creatorInfo.Username).
			SetPlatformID(creatorInfo.PlatformID).
			SetSource(source).
			OnConflict().
			UpdateNewValues().
			Exec(s.ctx)
	})

	if err != nil {
		return resource.EmptyCID, err
	}

	return cid, nil
}

func (s *Store) SaveRecord(metadata *models.Metadata, characterCard *png.CharacterCard, time timestamp.Nano, importIndex ...int) (resource.RID, error) {
	idx := 0
	if len(importIndex) > 0 {
		idx = importIndex[0]
	}

	if err := s.UpsertTags(metadata.Tags); err != nil {
		return 0, err
	}
	tids := slicesx.Map(metadata.Tags, func(tag models.Tag) resource.TID {
		return resource.TID(tag.Slug)
	})

	cid, err := s.UpsertCreator(metadata.Source, metadata.CreatorInfo)
	if err != nil {
		return 0, err
	}

	rid, err := s.client.RecordEntity.Create().
		SetImportTime(time).
		SetImportIndex(idx).
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
		SetGreetingsCount(metadata.GreetingsCount).
		SetIsFork(metadata.IsForked).
		SetCreatorID(cid).
		SetSyncStatus(resource.SyncSuccess).
		SetSyncTime(time).
		SetExportTime(timestamp.Nano(-1)).
		SetExportedVersion(timestamp.Nano(-1)).
		SetFavorite(false).
		OnConflictColumns(recordentity.FieldNormalizedURL).
		Update(func(u *ent.RecordEntityUpsert) {
			u.UpdateSource()
			u.UpdateNormalizedURL()
			u.UpdateDirectURL()
			u.UpdatePlatformID()
			u.UpdateCharacterID()
			u.UpdateName()
			u.UpdateTitle()
			u.UpdateTagline()
			u.UpdateCreateTime()
			u.UpdateUpdateTime()
			u.UpdateBookUpdateTime()
			u.UpdateGreetingsCount()
			u.UpdateIsFork()
			u.UpdateCreatorID()
			u.UpdateSyncStatus()
			u.UpdateSyncTime()
		}).
		ID(s.ctx)
	if err != nil {
		return 0, err
	}

	if err := s.upsertFTS(rid, characterCard); err != nil {
		return 0, err
	}

	return rid, s.client.RecordEntity.UpdateOneID(rid).
		ClearTags().
		AddTagIDs(tids...).
		Exec(s.ctx)
}

func (s *Store) RestoreRecord(rec *resource.Record, characterCard *png.CharacterCard) (resource.RID, error) {
	tags := slicesx.Map(rec.Tags, func(tag resource.Tag) models.Tag {
		return models.Tag{Slug: models.Slug(tag.ID), Name: tag.Name}
	})
	tids := slicesx.Map(rec.Tags, func(tag resource.Tag) resource.TID { return tag.ID })

	syncStatus := rec.SyncStatus
	if syncStatus == "" {
		syncStatus = resource.SyncSuccess
	}

	if err := s.UpsertTags(tags); err != nil {
		return 0, err
	}

	cid, err := s.UpsertCreator(rec.Creator.Source, models.CreatorInfo{
		Nickname:   rec.Creator.Nickname,
		Username:   rec.Creator.Username,
		PlatformID: rec.Creator.PlatformID,
	})
	if err != nil {
		return 0, err
	}

	rid, err := s.client.RecordEntity.Create().
		SetImportTime(rec.ImportTime).
		SetImportIndex(rec.ImportIndex).
		SetSource(rec.InfoData.Source).
		SetNormalizedURL(rec.NormalizedURL).
		SetDirectURL(rec.DirectURL).
		SetPlatformID(rec.InfoData.PlatformID).
		SetCharacterID(rec.CharacterID).
		SetName(rec.Name).
		SetTitle(rec.Title).
		SetTagline(rec.Tagline).
		SetCreateTime(rec.CreateTime).
		SetUpdateTime(rec.UpdateTime).
		SetBookUpdateTime(rec.BookUpdateTime).
		SetGreetingsCount(rec.GreetingsCount).
		SetIsFork(rec.IsFork).
		SetCreatorID(cid).
		SetSyncStatus(syncStatus).
		SetSyncTime(rec.SyncTime).
		SetExportTime(rec.ExportTime).
		SetExportedVersion(rec.ExportedVersion).
		SetFavorite(rec.Favorite).
		OnConflictColumns(recordentity.FieldNormalizedURL).
		UpdateNewValues().
		ID(s.ctx)
	if err != nil {
		return 0, err
	}

	if err := s.upsertFTS(rid, characterCard); err != nil {
		return 0, err
	}

	return rid, s.client.RecordEntity.UpdateOneID(rid).
		ClearTags().
		AddTagIDs(tids...).
		Exec(s.ctx)
}

func (s *Store) upsertFTS(rid resource.RID, characterCard *png.CharacterCard) error {
	_, err := withLock(s, true, func() (void, error) {
		_, err := s.client.ExecContext(s.ctx,
			`INSERT OR REPLACE INTO fts VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			rid,
			characterCard.Description,
			characterCard.Personality,
			characterCard.Scenario,
			characterCard.FirstMessage,
			characterCard.MessageExamples,
			characterCard.CreatorNotes,
			characterCard.SystemPrompt,
			characterCard.PostHistoryInstructions,
			stringsx.JoinNonBlank("\n", characterCard.AlternateGreetings...),
			characterCard.DepthPrompt.Prompt,
		)
		return void{}, err
	})
	return err
}

func (s *Store) UpdateSyncData(rid resource.RID, syncData resource.SyncData) error {
	_, err := withLock(s, true, func() (void, error) {
		return void{}, s.client.RecordEntity.UpdateOneID(rid).
			SetSyncStatus(syncData.SyncStatus).
			SetSyncTime(syncData.SyncTime).
			Exec(s.ctx)
	})
	return err
}

func (s *Store) UpdateExportData(rid resource.RID, exportData resource.ExportData) error {
	_, err := withLock(s, true, func() (void, error) {
		return void{}, s.client.RecordEntity.UpdateOneID(rid).
			SetExportTime(exportData.ExportTime).
			SetExportedVersion(exportData.ExportedVersion).
			Exec(s.ctx)
	})
	return err
}

func (s *Store) UpdateFavoriteData(favorite bool, rids ...resource.RID) error {
	_, err := withLock(s, true, func() (void, error) {
		return void{}, s.client.RecordEntity.Update().
			Where(recordentity.IDIn(rids...)).
			SetFavorite(favorite).
			Exec(s.ctx)
	})
	return err
}

func (s *Store) ToggleFavorite(rid resource.RID) error {
	_, err := withLock(s, true, func() (void, error) {
		return void{}, s.client.RecordEntity.UpdateOneID(rid).
			Modify(func(u *entsql.UpdateBuilder) {
				u.Set(recordentity.FieldFavorite, entsql.Expr("NOT "+recordentity.FieldFavorite))
			}).
			Exec(s.ctx)
	})
	return err
}

func (s *Store) Delete(rids ...resource.RID) (int, error) {
	return withLock(s, true, func() (int, error) {
		return s.client.RecordEntity.Delete().Where(recordentity.IDIn(rids...)).Exec(s.ctx)
	})
}

func (s *Store) CleanupCreators() (int, error) {
	return withLock(s, true, func() (int, error) {
		return s.client.CreatorEntity.
			Delete().
			Where(creatorentity.Not(creatorentity.HasRecords())).
			Exec(s.ctx)
	})
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

	s.mu.Lock()
	defer s.mu.Unlock()

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

func withLock[T any](s *Store, writing bool, op func() (T, error)) (T, error) {
	if s.isTransaction {
		return op()
	}
	if writing {
		s.mu.Lock()
		defer s.mu.Unlock()
	} else {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}
	return op()
}

func (s *Store) getTxStore(txClient *ent.Client) *Store {
	txStore := *s
	txStore.client = txClient
	txStore.isTransaction = true
	return &txStore
}
