package preferences

// Type is the configuration file format
type Type string

const (
	YAML Type = "yaml"
	JSON Type = "json"
)

// Options configures the preferences service
type Options struct {
	Path string
	Type Type
}

// Service manages application preferences
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
	GetBool(key Key) bool
	SetBool(key Key, value bool)
	RestoreDefaults()
	Save() error
}
