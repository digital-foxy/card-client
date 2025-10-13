package client

import (
	"time"

	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/preferences"
	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/blob/pblob"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/record/erecord"
	"github.com/r3dpixel/toolkit/reqx"
)

type Config struct {
	Http        reqx.Options
	Preferences preferences.Options
	Library     library.Options
}

func DefaultConfig() Config {
	return Config{
		Http: reqx.Options{
			RetryCount:    4,
			MinBackoff:    10 * time.Millisecond,
			MaxBackoff:    500 * time.Millisecond,
			EnableHttp3:   true,
			Impersonation: reqx.Chrome,
		},
		Preferences: preferences.Options{
			Path: "preferences",
			Type: preferences.YAML,
		},
		Library: library.Options{
			Path:          "vaults",
			MaxVaults:     50,
			RecordStorage: library.EntSQL,
			BlobStorage:   library.Pebble,
			RecordStorages: map[library.RecordStorage]record.Builder{
				library.EntSQL: erecord.Builder{CacheConnections: true, MaxConnections: 5, MaxIdleConnections: 2, MaxLifetime: 0},
			},
			BlobStorages: map[library.BlobStorage]blob.Builder{
				library.Pebble: pblob.Builder{MaxVersions: 5, ThumbnailSize: 256},
			},
		},
	}
}
