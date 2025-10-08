package opts

import (
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/toolkit/reqx"
)

type AppConfig struct {
	StoreOptions
	VaultOptions
	PreferencesOptions
	reqx.Options
}

type StoreOptions struct {
	record.Options
	PngOptions
}

type PngOptions struct {
	MaxVersions   int
	ThumbnailSize int
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
