package preferences

type Type string

const (
	YAML Type = "yaml"
	JSON Type = "json"
)

type Options struct {
	Path string
	Type Type
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
