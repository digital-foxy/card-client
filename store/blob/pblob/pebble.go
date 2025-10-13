package pblob

import (
	"io"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
)

type txDB interface {
	Get(key []byte) ([]byte, io.Closer, error)
	NewIter(o *pebble.IterOptions) (*pebble.Iterator, error)
	Set(key []byte, value []byte, opts *pebble.WriteOptions) error
	SingleDelete(key []byte, opts *pebble.WriteOptions) error
	DeleteRange(start, end []byte, opts *pebble.WriteOptions) error
}

type snapshotWrapper struct {
	*pebble.Snapshot
}

func (s *snapshotWrapper) Set(key []byte, value []byte, opts *pebble.WriteOptions) error {
	return nil
}

func (s *snapshotWrapper) SingleDelete(key []byte, opts *pebble.WriteOptions) error {
	return nil
}

func (s *snapshotWrapper) DeleteRange(start, end []byte, opts *pebble.WriteOptions) error {
	return nil
}

func defaultPebbleOpts() *pebble.Options {
	opts := &pebble.Options{
		// Disable logging
		Logger:          nil,
		LoggerAndTracer: nil,
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
