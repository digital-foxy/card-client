package badgerblob

import (
	"context"
	"encoding/binary"
	"image"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type storeKey struct {
	RID     resource.RID
	Version timestamp.Nanoh
}

type Options struct {
	Path string
}

type Store struct {
	db       *pebble.DB
	ctx      context.Context
	snapshot *pebble.Snapshot
	batch    *pebble.Batch
}

func New(opts Options) (blob.Store, error) {
	db, err := pebble.Open(opts.Path, defaultPebbleOpts())
	return &Store{db: db}, err
}

func defaultPebbleOpts() *pebble.Options {
	opts := &pebble.Options{
		// Cache for blob metadata and indexes
		Cache: pebble.NewCache(512 << 20), // 512MB
		// Memory table size - larger for big blobs
		MemTableSize:                64 << 20, // 64MB
		MemTableStopWritesThreshold: 4,
		// L0 settings for write throughput
		L0CompactionThreshold: 4,
		L0StopWritesThreshold: 12,
		// Base level size
		LBaseMaxBytes: 256 << 20, // 256MB
		// Last write wins - no crash recovery needed
		DisableWAL: true,
		// Max manifest size
		MaxManifestFileSize: 128 << 20,
	}
	// Configure levels - no compression, bloom filters
	for i := range opts.Levels {
		opts.Levels[i].Compression = func() *sstable.CompressionProfile { return sstable.NoCompression }
		opts.Levels[i].FilterPolicy = bloom.FilterPolicy(10)
		opts.Levels[i].FilterType = pebble.TableFilter
	}

	// Value separation for blobs (1-2MB typical, up to 30MB max)
	opts.Experimental.ValueSeparationPolicy = func() pebble.ValueSeparationPolicy {
		return pebble.ValueSeparationPolicy{
			Enabled:               true,
			MinimumSize:           128 << 10, // 128KB threshold
			MaxBlobReferenceDepth: 3,         // Low for better scan performance
			RewriteMinimumAge:     4 * time.Hour,
			TargetGarbageRatio:    0.4,
		}
	}

	opts.EnsureDefaults()
	return opts
}

func (s *Store) Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	key := compositeKey(rid, version)
	bytes, closer, err := s.db.Get(key[:])
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return png.FromBytes(bytes).First().Get()
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

func (s *Store) WithContext(ctx context.Context) blob.Store {
	//TODO implement me
	panic("implement me")
}

func (s *Store) WithTx(fn func(blob.Store) error) error {
	s.db.Get()
}

func (s *Store) Close() error {
	return s.db.Close()
}

type readTx struct {
	snapshot *pebble.Snapshot
}

func (r *readTx) Get(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	snp := r.snapshot.Get()
}

func (r *readTx) Thumbnail(rid resource.RID) (image.Image, error) {
	//TODO implement me
	panic("implement me")
}

func (r *readTx) Versions(rid resource.RID) []timestamp.Nano {
	//TODO implement me
	panic("implement me")
}
