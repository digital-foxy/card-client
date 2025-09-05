package preferences

import (
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/stringsx"
)

type ValueType byte

const (
	IntegerValue ValueType = iota
	StringValue
)

type KeyType byte

const (
	StandardKey KeyType = iota
	PathKey
)

type Key struct {
	ID           string
	Name         string
	KeyType      KeyType
	DefaultValue any
	ValueType    ValueType
}

var ExportPathKey = Key{
	ID:           "export_path",
	Name:         "Export Path",
	KeyType:      PathKey,
	ValueType:    StringValue,
	DefaultValue: filex.GetCWD(),
}
var MaxExportSizeKey = Key{
	ID:           "max_export_size",
	Name:         "Max Export Size",
	KeyType:      StandardKey,
	ValueType:    IntegerValue,
	DefaultValue: 3072,
}
var LastLoadedVaultKey = Key{
	ID:           "last_loaded_vault",
	Name:         "Last Vault",
	KeyType:      StandardKey,
	ValueType:    StringValue,
	DefaultValue: stringsx.Empty,
}

var Keys = []Key{
	ExportPathKey,
	MaxExportSizeKey,
}

type Service interface {
	Keys() []Key
	RegisterKey(key Key)
	GetAll() map[string]any
	SetAll(data map[string]any)
	Get(key Key) any
	Set(key Key, value any)
	GetString(key Key) string
	SetString(key Key, value string)
	GetInt(key Key) int
	SetInt(key Key, value int)
	Save() error
}
