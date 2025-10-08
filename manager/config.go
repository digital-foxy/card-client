package manager

import (
	"time"

	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/toolkit/reqx"
)

func DefaultConfig() opts.AppConfig {
	return opts.AppConfig{
		StoreOptions: opts.StoreOptions{
			DbOptions: opts.DbOptions{
				MaxConnections:  10,
				IdleConnections: 2,
				MaxLifetime:     0,
			},
			PngOptions: opts.PngOptions{
				MaxVersions:   5,
				ThumbnailSize: 256,
			},
		},
		VaultOptions: opts.VaultOptions{
			RootDir:    "vaults",
			VaultLimit: 50,
		},
		PreferencesOptions: opts.PreferencesOptions{
			FilePath: "preferences",
			FileType: opts.YAML,
		},
		ClientOptions: reqx.Options{
			RetryCount:    4,
			MinBackoff:    10 * time.Millisecond,
			MaxBackoff:    500 * time.Millisecond,
			EnableHttp3:   false,
			Impersonation: reqx.Chrome,
		},
	}
}
