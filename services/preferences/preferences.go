package preferences

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
	Id        string
	Name      string
	ValueType ValueType
	KeyType   KeyType
}

var ExportPathKey = Key{"export_path", "Export Path", StringValue, PathKey}
var MaxExportSizeKey = Key{"max_export_size", "Max Export Size", IntegerValue, StandardKey}
var LastLoadedVaultKey = Key{"last_loaded_vault", "Last Vault", StringValue, StandardKey}

var Keys = []Key{
	ExportPathKey,
	MaxExportSizeKey,
}

type Service interface {
	Keys() []Key
	RegisterKey(key Key, defaultValue any)
	GetAll() map[string]any
	SetAll(data map[string]any)
	Get(key string) any
	Set(key string, value any)
	GetString(key string) string
	SetString(key string, value string)
	GetInt(key string) int
	SetInt(key string, value int)
	Save() error
}
