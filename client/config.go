package client

import (
	"time"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/blob/pblob"
	"github.com/digital-foxy/card-client/store/blob/sqlblob"
	"github.com/digital-foxy/card-client/store/record"
	"github.com/digital-foxy/card-client/store/record/erecord"
	"github.com/digital-foxy/toolkit/reqx"
)

// Config holds all client configuration options
type Config struct {
	Http                reqx.Options
	Preferences         preferences.Options
	Library             library.Options
	UploadThumbnailSize int
	Workers             int
}

// DefaultConfig returns the default client configuration
func DefaultConfig() Config {
	return Config{
		Http: reqx.Options{
			RetryCount:        4,
			MinBackoff:        10 * time.Millisecond,
			MaxBackoff:        500 * time.Millisecond,
			DisableKeepAlives: true,
			Impersonation:     reqx.Chrome,
		},
		Preferences: preferences.Options{
			Path: "preferences",
			Type: preferences.YAML,
		},
		Library: library.Options{
			Path:          "vaults",
			MaxVaults:     50,
			RecordStorage: library.EntSQL,
			BlobStorage:   library.SQLiteBlob,
			RecordStorages: map[library.RecordStorage]record.Builder{
				library.EntSQL: erecord.Builder{CacheConnections: true, MaxConnections: 5, MaxIdleConnections: 2, MaxLifetime: 0},
			},
			BlobStorages: map[library.BlobStorage]blob.Builder{
				library.Pebble:     pblob.Builder{MaxVersions: 5, ThumbnailSize: 256},
				library.SQLiteBlob: sqlblob.Builder{MaxVersions: 5, ThumbnailSize: 256},
			},
		},
		UploadThumbnailSize: 256,
		Workers:             5,
	}
}
