package preferences

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/preferences"
	"github.com/stretchr/testify/assert"
)

func setupTestDirectory(t *testing.T) string {
	t.Helper()
	originalWD, err := os.Getwd()
	require.NoError(t, err)

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(originalWD)
		if err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	})
	return tempDir
}

func TestNewService_Initialization(t *testing.T) {
	t.Run("Creates a new config file if none exists", func(t *testing.T) {
		setupTestDirectory(t)
		configPath := filepath.Join(".", defaultFileName+".json")

		_, err := os.Stat(configPath)
		require.True(t, os.IsNotExist(err))

		s := NewService(opts.PreferencesOptions{FileType: "json"})
		require.NotNil(t, s)

		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		assert.NotEmpty(t, s.GetString(preferences.ExportPathKey.Id))
		assert.Empty(t, s.GetString(preferences.LastLoadedVaultKey.Id))
	})

	t.Run("Loads values from an existing config file", func(t *testing.T) {
		tempDir := setupTestDirectory(t)
		configPath := filepath.Join(tempDir, "my-prefs.json")
		configFileContent := `{"export_path": "/custom/path", "last_loaded_vault": "my-vault"}`
		require.NoError(t, os.WriteFile(configPath, []byte(configFileContent), 0644))

		s := NewService(opts.PreferencesOptions{FilePath: "my-prefs", FileType: "json"})

		assert.Equal(t, "/custom/path", s.GetString(preferences.ExportPathKey.Id))
		assert.Equal(t, "my-vault", s.GetString(preferences.LastLoadedVaultKey.Id))
	})
}

func TestService_GettersAndSetters(t *testing.T) {
	setupTestDirectory(t)
	s := NewService(opts.PreferencesOptions{FileType: "json"})

	s.Set("generic_key", "generic_value")
	assert.Equal(t, "generic_value", s.Get("generic_key"))

	s.SetString("string_key", "string_value")
	assert.Equal(t, "string_value", s.GetString("string_key"))

	s.SetInt("int_key", 123)
	assert.Equal(t, 123, s.GetInt("int_key"))

	newData := map[string]any{
		"new_string":                 "hello",
		"new_int":                    456,
		preferences.ExportPathKey.Id: "/override/path",
	}
	s.SetAll(newData)

	allSettings := s.GetAll()
	assert.Equal(t, "hello", allSettings["new_string"])
	assert.EqualValues(t, 456, allSettings["new_int"])
	assert.Equal(t, "/override/path", allSettings[preferences.ExportPathKey.Id])
}

func TestService_KeysAndRegisterKey(t *testing.T) {
	setupTestDirectory(t)
	s := NewService(opts.PreferencesOptions{FileType: "json"})
	initialKeyCount := len(s.Keys())

	newKey := preferences.Key{Id: "new_key", Name: "New Key", ValueType: preferences.IntegerValue}
	defaultValue := 999
	s.RegisterKey(newKey, defaultValue)

	assert.Len(t, s.Keys(), initialKeyCount+1)
	assert.Equal(t, defaultValue, s.GetInt("new_key"))
}

func TestService_Save(t *testing.T) {
	t.Run("Successfully saves config to a file", func(t *testing.T) {
		tempDir := setupTestDirectory(t)
		configPath := filepath.Join(tempDir, "preferences.json")
		s := NewService(opts.PreferencesOptions{FileType: "json"})

		s.SetString("user_name", "test-user")
		s.SetInt("login_attempts", 5)

		err := s.Save()
		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"user_name": "test-user"`)
		assert.Contains(t, string(content), `"login_attempts": 5`)
	})
}

func TestService_Concurrency(t *testing.T) {
	setupTestDirectory(t)
	s := NewService(opts.PreferencesOptions{FileType: "json"})
	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", n)
			value := fmt.Sprintf("value-%d", n)

			s.Set(key, value)
			s.Get(key)
			s.GetString(key)
			s.GetAll()

			if n%10 == 0 {
				_ = s.Save()
			}
		}(i)
	}

	wg.Wait()

	allSettings := s.GetAll()
	assert.Len(t, allSettings, numGoroutines+3)
}
