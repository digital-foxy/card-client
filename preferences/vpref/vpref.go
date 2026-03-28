package vpref

import (
	"slices"
	"sync"

	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/toolkit/stringsx"
	"github.com/digital-foxy/toolkit/trace"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	defaultFileName = "preferences"
)

// Preferences implements preferences.Service using Viper
type Preferences struct {
	dataMutex sync.RWMutex
	keys      []preferences.Key
	viper     *viper.Viper
}

// NewService creates a new Viper-based preferences service
func NewService(opts preferences.Options) *Preferences {
	s := Preferences{
		keys:  slices.Clone(preferences.Keys),
		viper: viper.New(),
	}

	// Set default values for all keys
	for _, key := range s.keys {
		s.viper.SetDefault(key.ID, key.DefaultValue)
	}

	if stringsx.IsBlank(opts.Path) {
		opts.Path = defaultFileName
	}
	s.readConfigFile(opts.Path, opts.Type)
	return &s
}

func (s *Preferences) readConfigFile(fileName string, fileType preferences.Type) {
	s.viper.SetConfigName(fileName)
	s.viper.AddConfigPath(".")
	s.viper.SetConfigType(string(fileType))
	err := s.viper.ReadInConfig()
	if err != nil {
		_ = s.viper.SafeWriteConfig()
		log.Warn().Err(err).
			Str(trace.SERVICE, "preferences").
			Msg("Failed to read config. Loading default values.")
	}
}

func (s *Preferences) Keys() []preferences.Key {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.keys
}

func (s *Preferences) RegisterKey(key preferences.Key) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.keys = append(s.keys, key)
	s.viper.SetDefault(key.ID, key.DefaultValue)
}

func (s *Preferences) RegisterKeys(keys ...preferences.Key) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.keys = append(s.keys, keys...)
	for _, key := range s.keys {
		s.viper.SetDefault(key.ID, key.DefaultValue)
	}
}

func (s *Preferences) GetAll() map[string]any {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.AllSettings()
}

func (s *Preferences) SetAll(data map[string]any) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	_ = s.viper.MergeConfigMap(data)
}

func (s *Preferences) Get(key preferences.Key) any {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.Get(key.ID)
}

func (s *Preferences) Set(key preferences.Key, value any) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.viper.Set(key.ID, value)
}

func (s *Preferences) GetString(key preferences.Key) string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.GetString(key.ID)
}

func (s *Preferences) SetString(key preferences.Key, value string) {
	s.Set(key, value)
}

func (s *Preferences) GetInt(key preferences.Key) int {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.GetInt(key.ID)
}

func (s *Preferences) SetInt(key preferences.Key, value int) {
	s.Set(key, value)
}

func (s *Preferences) GetBool(key preferences.Key) bool {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.GetBool(key.ID)
}

func (s *Preferences) SetBool(key preferences.Key, value bool) {
	s.Set(key, value)
}

func (s *Preferences) RestoreDefaults() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	for _, key := range s.keys {
		s.viper.Set(key.ID, key.DefaultValue)
	}
}

func (s *Preferences) Save() error {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	if err := s.viper.WriteConfig(); err != nil {
		log.Warn().Err(err).
			Str(trace.SERVICE, "preferences").
			Msg("Failed to write config file")
		return err
	}

	return nil
}
