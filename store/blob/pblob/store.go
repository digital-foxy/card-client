package pblob

import (
	"bytes"
	"context"
	"image"
	"slices"

	"github.com/cockroachdb/pebble/v2"
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
)

// Builder implements blob.Builder using Pebble
type Builder Options

func (b Builder) Build(path string) (blob.Store, error) {
	return New(path, Options(b))
}

// Options configures the Pebble blob store
type Options struct {
	MaxVersions   int
	ThumbnailSize int
}

// Store implements blob.Store using Pebble KV
type Store struct {
	maxVersions   int
	thumbnailSize int
	db            *pebble.DB
	txDB          txDB
	ctx           context.Context
	isTransaction bool
}

// New creates a new Pebble blob store at the given path
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

func (s *Store) RIDs() ([]resource.RID, error) {
	iter, err := s.txDB.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	iter.First()

	if !iter.Valid() {
		return nil, nil
	}

	rid := keyOf(iter.Key()).GetRID()
	rids := []resource.RID{rid}

	for iter.First(); iter.Valid(); iter.Next() {
		currentRid := keyOf(iter.Key()).GetRID()
		if rid != currentRid {
			rid = currentRid
			rids = append(rids, rid)
		}
	}

	return rids, nil
}

func (s *Store) GetCharacterCard(rid resource.RID, version timestamp.Nano) (*png.CharacterCard, error) {
	rawCard, err := s.GetRawCard(rid, version)
	if err != nil {
		return nil, err
	}
	return rawCard.Decode()
}

func (s *Store) GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	b, closer, err := s.txDB.Get(newKey().RID(rid).Type(pngType).Version(version).Bytes())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return png.FromBytes(b).First().Get()
}

func (s *Store) GetRawCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	b, closer, err := s.txDB.Get(newKey().RID(rid).Type(pngType).Version(version).Bytes())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return slices.Clone(b), nil
}

func (s *Store) GetSheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error) {
	cc, err := s.GetCharacterCard(rid, version)
	if err != nil {
		return nil, err
	}
	return cc.Sheet, nil
}

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

func (s *Store) Thumbnail(rid resource.RID) (image.Image, error) {
	b, closer, err := s.txDB.Get(newKey().RID(rid).Type(thumbnailType).Bytes())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return imgconv.Decode(bytes.NewReader(b))
}

func (s *Store) ThumbnailBytes(rid resource.RID) ([]byte, error) {
	b, closer, err := s.txDB.Get(newKey().RID(rid).Type(thumbnailType).Bytes())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return slices.Clone(b), nil
}

func (s *Store) Versions(rid resource.RID) []timestamp.Nano {
	iter, err := s.txDB.NewIter(&pebble.IterOptions{
		LowerBound: newKey().RID(rid).Type(pngType).Bytes(),
		UpperBound: newKey().RID(rid).Type(pngType + 1).Bytes(),
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
	return s.keyExists(newKey().RID(rid).Type(pngType).Version(version).Bytes())
}

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

	pngItemKey := newKey().RID(rid).Type(pngType).Version(version).Bytes()
	exists, err := s.keyExists(pngItemKey)
	if err != nil {
		return err
	}

	if !exists {
		if err := s.deleteOldVersions(rid); err != nil {
			return err
		}
	}

	thumbnail, err := rawJsonCard.Thumbnail(s.thumbnailSize)
	if err != nil {
		return err
	}

	t, err := imagex.ToBytes(thumbnail, imgconv.WEBP)
	if err != nil {
		return err
	}

	if err := s.txDB.Set(newKey().RID(rid).Type(thumbnailType).Bytes(), t, pebble.NoSync); err != nil {
		return err
	}

	b, err := rawJsonCard.ToRaw().ToBytes()
	if err != nil {
		return err
	}
	return s.txDB.Set(pngItemKey, b, pebble.NoSync)
}

func (s *Store) DeleteVersion(rid resource.RID, version timestamp.Nano) error {
	return s.WithTx(func(store blob.TxStore) error {
		return store.(*Store).internalDeleteVersion(rid, version)
	})
}

func (s *Store) internalDeleteVersion(rid resource.RID, version timestamp.Nano) error {
	return s.txDB.SingleDelete(newKey().RID(rid).Type(pngType).Version(version).Bytes(), pebble.Sync)
}

func (s *Store) DeleteVersions(rid resource.RID, lower timestamp.Nano, upper timestamp.Nano) error {
	return s.WithTx(func(store blob.TxStore) error {
		return store.(*Store).internalDeleteVersions(rid, lower, upper)
	})
}

func (s *Store) internalDeleteVersions(rid resource.RID, lower timestamp.Nano, upper timestamp.Nano) error {
	return s.txDB.DeleteRange(
		newKey().RID(rid).Type(pngType).Version(lower).Bytes(),
		newKey().RID(rid).Type(pngType).Version(upper).Bytes(),
		pebble.Sync,
	)
}

func (s *Store) Delete(rid resource.RID) error {
	return s.txDB.DeleteRange(
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
	return s.WithTx(func(store blob.TxStore) error {
		storeTx := store.(*Store)
		versions := storeTx.Versions(rid)
		versionCount := len(versions)

		if versionCount < storeTx.maxVersions {
			return nil
		}

		deleteUpTo := versions[versionCount-storeTx.maxVersions+1]
		return storeTx.DeleteVersions(rid, 0, deleteUpTo)
	})
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
