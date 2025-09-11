package badgerblob

import (
	"image"
	"path"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

const (
	dbFile string = "bdg.cards"
)

type Store struct {
	db *badger.DB
}

func New(vaultPath string) *Store {
	dbPath := path.Join(vaultPath, dbFile)
	opts := badger.DefaultOptions(dbPath)
	opts.MetricsEnabled = false
	opts.SyncWrites = false
	opts.Logger = nil
	opts.MemTableSize = 128 << 20
	opts.Compression = options.None
	opts.ValueThreshold = 1024
	opts.ValueLogFileSize = 1 << 30
	opts.VerifyValueChecksum = false
	opts.DetectConflicts = false

	return &Store{}
}

func (s *Store) Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Store) LoadThumbnail(rid resource.RID) (image.Image, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Store) Versions(rid resource.RID) []timestamp.Nano {
	//TODO implement me
	panic("implement me")
}

func (s *Store) Put(rid resource.RID, version timestamp.Nano, rawCard *png.RawCard) error {
	//TODO implement me
	panic("implement me")
}

func (s *Store) DeleteVersion(rid resource.RID, version timestamp.Nano) error {
	//TODO implement me
	panic("implement me")
}

func (s *Store) Delete(rid resource.RID) error {
	//TODO implement me
	panic("implement me")
}
