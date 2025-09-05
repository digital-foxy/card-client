package preferences

import (
	"os"
	"slices"
	"sync"

	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/preferences"
	"github.com/r3dpixel/toolkit/stringsx"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	defaultFileName      = "preferences"
	defaultMaxExportSize = 3072
)

type Service struct {
	dataMutex sync.RWMutex
	keys      []preferences.Key
	viper     *viper.Viper
}

func NewService(opts opts.PreferencesOptions) *Service {
	s := Service{
		keys:  slices.Clone(preferences.Keys),
		viper: viper.New(),
	}
	if stringsx.IsBlank(opts.FilePath) {
		opts.FilePath = defaultFileName
	}
	s.setDefaults()
	s.readConfigFile(opts.FilePath, opts.FileType)
	return &s
}

func (s *Service) setDefaults() {
	currentDirectory, err := os.Getwd()
	if err != nil {
		currentDirectory = "."
	}
	s.viper.SetDefault(preferences.ExportPathKey.Id, currentDirectory)
	s.viper.SetDefault(preferences.LastLoadedVaultKey.Id, "")
	s.viper.SetDefault(preferences.MaxExportSizeKey.Id, defaultMaxExportSize)
}
func (s *Service) readConfigFile(fileName string, fileType opts.ConfigType) {
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

func (s *Service) GetAll() map[string]any {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.AllSettings()
}

func (s *Service) SetAll(data map[string]any) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	_ = s.viper.MergeConfigMap(data)
}

func (s *Service) Get(key string) any {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.Get(key)
}

func (s *Service) Set(key string, value any) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.viper.Set(key, value)
}

func (s *Service) GetString(key string) string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.GetString(key)
}

func (s *Service) SetString(key string, value string) {
	s.Set(key, value)
}

func (s *Service) GetInt(key string) int {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.viper.GetInt(key)
}

func (s *Service) SetInt(key string, value int) {
	s.Set(key, value)
}

func (s *Service) Save() error {
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

func (s *Service) RegisterKey(key preferences.Key, defaultValue any) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.keys = append(s.keys, key)
	s.viper.SetDefault(key.Id, defaultValue)
}

func (s *Service) Keys() []preferences.Key {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.keys
}
