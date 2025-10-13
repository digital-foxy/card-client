package pblob

import (
	"bytes"
	"context"
	"image"
	"slices"

	"github.com/cockroachdb/pebble/v2"
	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/imagex"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/sunshineplan/imgconv"
)

const (
	defaultMaxVersions   = 5
	defaultThumbnailSize = 256
)

type Builder Options

func (b Builder) Build(path string) (blob.Store, error) {
	return New(path, Options(b))
}

type Options struct {
	MaxVersions   int
	ThumbnailSize int
}

type Store struct {
	maxVersions   int
	thumbnailSize int
	db            *pebble.DB
	txDB          txDB
	ctx           context.Context
	isTransaction bool
}

func New(path string, opts Options) (blob.Store, error) {
	if opts.MaxVersions <= 0 {
		opts.MaxVersions = defaultMaxVersions
	}

	if opts.ThumbnailSize <= 0 {
		opts.ThumbnailSize = defaultThumbnailSize
	}

	db, err := pebble.Open(path, defaultPebbleOpts())
	return &Store{
		maxVersions:   opts.MaxVersions,
		thumbnailSize: opts.ThumbnailSize,
		db:            db,
		txDB:          db,
		ctx:           context.Background(),
		isTransaction: false,
	}, err
}

func (s *Store) Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	b, closer, err := s.txDB.Get(versionKey(rid, version))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return png.FromBytes(b).First().Get()
}

func (s *Store) GetBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	b, closer, err := s.txDB.Get(versionKey(rid, version))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return slices.Clone(b), nil
}

func (s *Store) Thumbnail(rid resource.RID) (image.Image, error) {
	b, closer, err := s.txDB.Get(thumbnailKey(rid))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return imgconv.Decode(bytes.NewReader(b))
}

func (s *Store) ThumbnailBytes(rid resource.RID) ([]byte, error) {
	b, closer, err := s.txDB.Get(thumbnailKey(rid))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return slices.Clone(b), nil
}

func (s *Store) Versions(rid resource.RID) []timestamp.Nano {
	prefix := versionPrefix(rid)
	iter, err := s.txDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: thumbnailKey(rid),
	})
	if err != nil {
		return nil
	}
	defer iter.Close()

	var versionsList []timestamp.Nano
	for iter.First(); iter.Valid(); iter.Next() {
		if err := s.ctx.Err(); err != nil {
			return nil
		}
		versionsList = append(versionsList, keyOf(iter.Key()).GetVersion())
	}
	return versionsList
}

func (s *Store) VersionExists(rid resource.RID, version timestamp.Nano) (bool, error) {
	return s.keyExists(versionKey(rid, version))
}

func (s *Store) Put(rid resource.RID, version timestamp.Nano, rawCard *png.RawCard) error {
	itemKey := versionKey(rid, version)
	exists, err := s.keyExists(itemKey)
	if err != nil {
		return err
	}

	if !exists {
		if err := s.deleteOldVersions(rid); err != nil {
			return err
		}
	}

	thumbnail, err := rawCard.Thumbnail(s.thumbnailSize)
	if err != nil {
		return err
	}

	t, err := imagex.ToBytes(thumbnail, imgconv.PNG)
	if err != nil {
		return err
	}

	if err := s.txDB.Set(thumbnailKey(rid), t, pebble.Sync); err != nil {
		return err
	}

	b, err := rawCard.ToBytes()
	if err != nil {
		return err
	}
	return s.txDB.Set(itemKey, b, pebble.Sync)
}

func (s *Store) DeleteVersion(rid resource.RID, version timestamp.Nano) error {
	return s.db.SingleDelete(versionKey(rid, version), pebble.Sync)
}

func (s *Store) DeleteVersions(rid resource.RID, lower timestamp.Nano, upper timestamp.Nano) error {
	return s.db.DeleteRange(
		newKey().RID(rid).Type(versionType).Version(lower).Bytes(),
		newKey().RID(rid).Type(versionType).Version(upper).Bytes(),
		pebble.Sync,
	)
}

func (s *Store) Delete(rid resource.RID) error {
	return s.db.DeleteRange(
		newKey().RID(rid).Type(minType).Bytes(),
		newKey().RID(rid).Type(maxType).Bytes(),
		pebble.Sync,
	)
}

func (s *Store) WithContext(ctx context.Context) blob.CtxStore {
	ctxStore := *s
	ctxStore.ctx = ctx
	return &ctxStore
}

func (s *Store) WithTx(fn func(store blob.TxStore) error) error {
	if s.isTransaction {
		return fn(s)
	}

	indexedBatch := s.db.NewIndexedBatch()
	defer indexedBatch.Close()

	if err := fn(s.getTxStore(indexedBatch)); err != nil {
		return err
	}

	return indexedBatch.Commit(pebble.Sync)
}

func (s *Store) WithReadTx(fn func(store blob.TxReadStore) error) error {
	if s.isTransaction {
		return fn(s)
	}

	snapshot := s.db.NewSnapshot()
	defer snapshot.Close()

	return fn(s.getTxStore(&snapshotWrapper{snapshot}))
}

func (s *Store) WithWriteTx(fn func(store blob.TxWriteStore) error) error {
	if s.isTransaction {
		return fn(s)
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	if err := fn(s.getTxStore(batch)); err != nil {
		return err
	}

	return batch.Commit(pebble.Sync)
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) deleteOldVersions(rid resource.RID) error {
	versions := s.Versions(rid)
	versionCount := len(versions)

	if versionCount < s.maxVersions {
		return nil
	}

	deleteUpTo := versions[versionCount-s.maxVersions+1]
	return s.DeleteVersions(rid, 0, deleteUpTo)
}

func (s *Store) keyExists(itemKey []byte) (bool, error) {
	iter, err := s.txDB.NewIter(&pebble.IterOptions{
		LowerBound: itemKey,
	})
	if err != nil {
		return false, err
	}
	defer iter.Close()

	return iter.First() && bytes.Equal(iter.Key(), itemKey), nil
}

func (s *Store) getTxStore(txDB txDB) *Store {
	txStore := *s
	txStore.txDB = txDB
	txStore.isTransaction = true
	return &txStore
}
