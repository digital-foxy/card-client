package opts

import (
	"time"

	"github.com/r3dpixel/toolkit/reqx"
)

type AppConfig struct {
	StoreOptions
	VaultOptions
	PreferencesOptions
	reqx.ClientOptions
}

type StoreOptions struct {
	DbOptions
	PngOptions
}

type PngOptions struct {
	MaxVersions   int
	ThumbnailSize int
}

type DbOptions struct {
	MaxConnections  int
	IdleConnections int
	MaxLifetime     time.Duration
}

type VaultOptions struct {
	RootDir    string
	VaultLimit int
}

type PreferencesOptions struct {
	FilePath string
	FileType ConfigType
}

type ConfigType string

const (
	YAML ConfigType = "yaml"
	JSON ConfigType = "json"
	TOML ConfigType = "toml"
)
