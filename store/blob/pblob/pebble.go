package pblob

import (
	"io"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
)

// txDB abstracts Pebble DB operations for transactions
type txDB interface {
	Get(key []byte) ([]byte, io.Closer, error)
	NewIter(o *pebble.IterOptions) (*pebble.Iterator, error)
	Set(key []byte, value []byte, opts *pebble.WriteOptions) error
	SingleDelete(key []byte, opts *pebble.WriteOptions) error
	DeleteRange(start, end []byte, opts *pebble.WriteOptions) error
}

type noopLogger struct{}

func (noopLogger) Infof(format string, args ...interface{})  {}
func (noopLogger) Errorf(format string, args ...interface{}) {}
func (noopLogger) Fatalf(format string, args ...interface{}) {}

// snapshotWrapper wraps Pebble snapshot for read-only transactions
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
		FormatMajorVersion: pebble.FormatNewest,
		Logger:             noopLogger{},
		// Cache for blob metadata and indexes
		Cache: pebble.NewCache(256 << 20),
		// Memory table size - larger for bigger flushes
		MemTableSize:                128 << 20,
		MemTableStopWritesThreshold: 4,
		// L0 settings - less aggressive to allow larger files
		L0CompactionThreshold: 2,
		L0StopWritesThreshold: 4,
		// Base level size - larger for bigger compactions
		LBaseMaxBytes: 2 << 30,
		// Max manifest size
		MaxManifestFileSize: 128 << 20,
	}
	// Target 128MB SST files at L0, doubles for each level
	opts.TargetFileSizes[0] = 128 << 20
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
			MaxBlobReferenceDepth: 3,
			RewriteMinimumAge:     5 * time.Minute,
			TargetGarbageRatio:    0.1,
		}
	}

	return opts
}
