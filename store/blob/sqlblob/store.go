package sqlblob

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/imagex"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/sunshineplan/imgconv"
)

const (
	defaultMaxVersions   = 5
	defaultThumbnailSize = 256
	driverName           = "sqlite3"
)

// Builder implements blob.Builder using SQLite
type Builder Options

func (b Builder) Build(path string) (blob.Store, error) {
	return New(path, Options(b))
}

// Options configures the SQLite blob store
type Options struct {
	MaxVersions   int
	ThumbnailSize int
}

// executor abstracts sql.DB and sql.Tx for shared query execution
type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store implements blob.Store using SQLite
type Store struct {
	db            *sql.DB
	exec          executor
	mu            *sync.RWMutex
	maxVersions   int
	thumbnailSize int
	ctx           context.Context
	isTransaction bool
}

// New creates a new SQLite blob store at the given path
func New(path string, opts Options) (blob.Store, error) {
	if opts.MaxVersions <= 0 {
		opts.MaxVersions = defaultMaxVersions
	}
	if opts.ThumbnailSize <= 0 {
		opts.ThumbnailSize = defaultThumbnailSize
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_fk=1", path)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(2)

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cards (
			rid     INTEGER NOT NULL,
			version INTEGER NOT NULL,
			png     BLOB NOT NULL,
			PRIMARY KEY (rid, version)
		);
		CREATE TABLE IF NOT EXISTS thumbnails (
			rid   INTEGER PRIMARY KEY,
			thumb BLOB NOT NULL
		);
	`); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		db:            db,
		exec:          db,
		mu:            &sync.RWMutex{},
		maxVersions:   opts.MaxVersions,
		thumbnailSize: opts.ThumbnailSize,
		ctx:           context.Background(),
		isTransaction: false,
	}, nil
}

// RIDs returns all unique RIDs in the store
func (s *Store) RIDs() ([]resource.RID, error) {
	return withReadLock(s, func() ([]resource.RID, error) {
		return queryList[resource.RID](s, "SELECT DISTINCT rid FROM cards ORDER BY rid")
	})
}

// GetCharacterCard returns a decoded CharacterCard from the PNG blob
func (s *Store) GetCharacterCard(rid resource.RID, version timestamp.Nano) (*png.CharacterCard, error) {
	rawCard, err := s.GetRawCard(rid, version)
	if err != nil {
		return nil, err
	}
	return rawCard.Decode()
}

// GetRawCard returns a parsed RawCard from the PNG blob
func (s *Store) GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	b, err := s.GetRawCardBytes(rid, version)
	if err != nil {
		return nil, err
	}
	return png.FromBytes(b).First().Get()
}

// GetRawCardBytes returns the raw PNG bytes for a card
func (s *Store) GetRawCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	return withReadLock(s, func() ([]byte, error) {
		var b []byte
		err := s.exec.QueryRowContext(s.ctx,
			"SELECT png FROM cards WHERE rid = ? AND version = ?", rid, version,
		).Scan(&b)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
}

// GetSheet returns the character Sheet extracted from the PNG blob
func (s *Store) GetSheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error) {
	rawCard, err := s.GetRawCard(rid, version)
	if err != nil {
		return nil, err
	}
	cc, err := rawCard.Decode()
	if err != nil {
		return nil, err
	}
	return cc.Sheet, nil
}

// GetSheetBytes returns the raw JSON bytes extracted from the PNG blob
func (s *Store) GetSheetBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	rawCard, err := s.GetRawCard(rid, version)
	if err != nil {
		return nil, err
	}
	rjc, err := rawCard.ToRawJson()
	if err != nil {
		return nil, err
	}
	return rjc.RawJsonData, nil
}

// Thumbnail returns the decoded thumbnail image for a card
func (s *Store) Thumbnail(rid resource.RID) (image.Image, error) {
	b, err := s.ThumbnailBytes(rid)
	if err != nil {
		return nil, err
	}
	return imgconv.Decode(bytes.NewReader(b))
}

// ThumbnailBytes returns the raw WebP thumbnail bytes for a card
func (s *Store) ThumbnailBytes(rid resource.RID) ([]byte, error) {
	return withReadLock(s, func() ([]byte, error) {
		var b []byte
		err := s.exec.QueryRowContext(s.ctx,
			"SELECT thumb FROM thumbnails WHERE rid = ?", rid,
		).Scan(&b)
		if err != nil {
			return nil, err
		}
		return b, nil
	})
}

// Versions returns all version timestamps for a card, sorted ascending
func (s *Store) Versions(rid resource.RID) []timestamp.Nano {
	versions, _ := withReadLock(s, func() ([]timestamp.Nano, error) {
		return queryList[timestamp.Nano](s, "SELECT version FROM cards WHERE rid = ? ORDER BY version ASC", rid)
	})
	return versions
}

// VersionExists checks if a specific version exists for a card
func (s *Store) VersionExists(rid resource.RID, version timestamp.Nano) (bool, error) {
	return withReadLock(s, func() (bool, error) {
		var exists int
		err := s.exec.QueryRowContext(s.ctx,
			"SELECT 1 FROM cards WHERE rid = ? AND version = ? LIMIT 1", rid, version,
		).Scan(&exists)
		if err == sql.ErrNoRows {
			return false, nil
		}
		return err == nil, err
	})
}

// Put stores a CharacterCard as PNG blob + thumbnail, pruning old versions
func (s *Store) Put(rid resource.RID, version timestamp.Nano, characterCard *png.CharacterCard) error {
	return s.WithTx(func(store blob.TxStore) error {
		return store.(*Store).internalPut(rid, version, characterCard)
	})
}

func (s *Store) internalPut(rid resource.RID, version timestamp.Nano, characterCard *png.CharacterCard) error {
	rawJsonCard, err := characterCard.ToRawJson()
	if err != nil {
		return err
	}

	// Generate thumbnail
	thumbnail, err := rawJsonCard.Thumbnail(s.thumbnailSize)
	if err != nil {
		return err
	}

	thumbBytes, err := imagex.ToBytes(thumbnail, imgconv.WEBP)
	if err != nil {
		return err
	}

	// Store thumbnail (upsert, unversioned)
	if _, err := s.exec.ExecContext(s.ctx,
		"INSERT OR REPLACE INTO thumbnails (rid, thumb) VALUES (?, ?)",
		rid, thumbBytes,
	); err != nil {
		return err
	}

	// Convert to PNG bytes
	pngBytes, err := rawJsonCard.ToRaw().ToBytes()
	if err != nil {
		return err
	}

	// Store PNG blob
	if _, err := s.exec.ExecContext(s.ctx,
		"INSERT OR REPLACE INTO cards (rid, version, png) VALUES (?, ?, ?)",
		rid, version, pngBytes,
	); err != nil {
		return err
	}

	// Prune old versions
	return s.deleteOldVersions(rid)
}

// DeleteVersion deletes a specific version of a card
func (s *Store) DeleteVersion(rid resource.RID, version timestamp.Nano) error {
	return s.WithTx(func(store blob.TxStore) error {
		_, err := store.(*Store).exec.ExecContext(store.(*Store).ctx,
			"DELETE FROM cards WHERE rid = ? AND version = ?", rid, version,
		)
		return err
	})
}

// DeleteVersions deletes a range of versions for a card [lower, upper)
func (s *Store) DeleteVersions(rid resource.RID, lower timestamp.Nano, upper timestamp.Nano) error {
	return s.WithTx(func(store blob.TxStore) error {
		_, err := store.(*Store).exec.ExecContext(store.(*Store).ctx,
			"DELETE FROM cards WHERE rid = ? AND version >= ? AND version < ?", rid, lower, upper,
		)
		return err
	})
}

// Delete removes all data for a card (all versions + thumbnail)
func (s *Store) Delete(rid resource.RID) error {
	return s.WithTx(func(store blob.TxStore) error {
		st := store.(*Store)
		if _, err := st.exec.ExecContext(st.ctx, "DELETE FROM cards WHERE rid = ?", rid); err != nil {
			return err
		}
		_, err := st.exec.ExecContext(st.ctx, "DELETE FROM thumbnails WHERE rid = ?", rid)
		return err
	})
}

// WithContext returns a context-aware copy of the store
func (s *Store) WithContext(ctx context.Context) blob.CtxStore {
	ctxStore := *s
	ctxStore.ctx = ctx
	return &ctxStore
}

// WithTx runs a function within a read-write transaction
func (s *Store) WithTx(fn func(store blob.TxStore) error) error {
	if s.isTransaction {
		return fn(s)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(s.ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(s.getTxStore(tx)); err != nil {
		return err
	}

	return tx.Commit()
}

// WithReadTx runs a function within a read-only context
func (s *Store) WithReadTx(fn func(store blob.TxReadStore) error) error {
	if s.isTransaction {
		return fn(s)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn(s)
}

// WithWriteTx runs a function within a write transaction
func (s *Store) WithWriteTx(fn func(store blob.TxWriteStore) error) error {
	if s.isTransaction {
		return fn(s)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(s.ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(s.getTxStore(tx)); err != nil {
		return err
	}

	return tx.Commit()
}

// Close closes the underlying database connection
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) deleteOldVersions(rid resource.RID) error {
	versions := s.Versions(rid)
	if len(versions) <= s.maxVersions {
		return nil
	}

	deleteUpTo := versions[len(versions)-s.maxVersions]
	_, err := s.exec.ExecContext(s.ctx,
		"DELETE FROM cards WHERE rid = ? AND version < ?", rid, deleteUpTo,
	)
	return err
}

func (s *Store) getTxStore(tx *sql.Tx) *Store {
	txStore := *s
	txStore.exec = tx
	txStore.isTransaction = true
	return &txStore
}

func queryList[T any](s *Store, query string, args ...any) ([]T, error) {
	rows, err := s.exec.QueryContext(s.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []T
	for rows.Next() {
		var v T
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func withReadLock[T any](s *Store, op func() (T, error)) (T, error) {
	if s.isTransaction {
		return op()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return op()
}
