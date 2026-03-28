package vpref

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/digital-foxy/card-client/preferences"
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
	testCases := []struct {
		name         string
		setupFunc    func(*testing.T) string
		expectError  bool
		validateFunc func(*testing.T, *Preferences)
	}{
		{
			name: "Creates a new config file if none exists",
			setupFunc: func(t *testing.T) string {
				tempDir := setupTestDirectory(t)
				configPath := filepath.Join(".", defaultFileName+".json")
				_, err := os.Stat(configPath)
				require.True(t, os.IsNotExist(err))
				return tempDir
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *Preferences) {
				configPath := filepath.Join(".", defaultFileName+".json")
				_, err := os.Stat(configPath)
				assert.NoError(t, err)

				assert.NotEmpty(t, s.GetString(preferences.ExportPathKey))
				assert.Empty(t, s.GetString(preferences.LastLoadedVaultKey))
			},
		},
		{
			name: "Loads values from an existing config file",
			setupFunc: func(t *testing.T) string {
				tempDir := setupTestDirectory(t)
				configPath := filepath.Join(tempDir, "my-prefs.json")
				configFileContent := `{"export_path": "/custom/path", "last_loaded_vault": "my-vault"}`
				require.NoError(t, os.WriteFile(configPath, []byte(configFileContent), 0644))
				return tempDir
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *Preferences) {
				assert.Equal(t, "/custom/path", s.GetString(preferences.ExportPathKey))
				assert.Equal(t, "my-vault", s.GetString(preferences.LastLoadedVaultKey))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFunc(t)

			var s *Preferences
			if tc.name == "Creates a new config file if none exists" {
				s = NewService(preferences.Options{Type: preferences.JSON})
			} else {
				s = NewService(preferences.Options{Path: "my-prefs", Type: preferences.JSON})
			}

			require.NotNil(t, s)
			tc.validateFunc(t, s)
		})
	}
}

func TestService_GettersAndSetters(t *testing.T) {
	setupTestDirectory(t)
	s := NewService(preferences.Options{Type: preferences.JSON})

	genericKey := preferences.Key{ID: "generic_key"}
	s.Set(genericKey, "generic_value")
	assert.Equal(t, "generic_value", s.Get(genericKey))

	stringKey := preferences.Key{ID: "string_key"}
	s.SetString(stringKey, "string_value")
	assert.Equal(t, "string_value", s.GetString(stringKey))

	intKey := preferences.Key{ID: "int_key"}
	s.SetInt(intKey, 123)
	assert.Equal(t, 123, s.GetInt(intKey))

	newData := map[string]any{
		"new_string":                 "hello",
		"new_int":                    456,
		preferences.ExportPathKey.ID: "/override/path",
	}
	s.SetAll(newData)

	allSettings := s.GetAll()
	assert.Equal(t, "hello", allSettings["new_string"])
	assert.EqualValues(t, 456, allSettings["new_int"])
	assert.Equal(t, "/override/path", allSettings[preferences.ExportPathKey.ID])
}

func TestService_KeysAndRegisterKey(t *testing.T) {
	setupTestDirectory(t)
	s := NewService(preferences.Options{Type: preferences.JSON})
	initialKeyCount := len(s.Keys())

	newKey := preferences.Key{ID: "new_key", Name: "New Key", ValueType: preferences.IntegerValue, DefaultValue: 999}
	s.RegisterKey(newKey)

	assert.Len(t, s.Keys(), initialKeyCount+1)
	assert.Equal(t, 999, s.GetInt(newKey))
}

func TestService_Save(t *testing.T) {
	t.Run("Successfully saves config to a file", func(t *testing.T) {
		tempDir := setupTestDirectory(t)
		configPath := filepath.Join(tempDir, "preferences.json")
		s := NewService(preferences.Options{Type: preferences.JSON})

		userNameKey := preferences.Key{ID: "user_name"}
		loginAttemptsKey := preferences.Key{ID: "login_attempts"}
		s.SetString(userNameKey, "test-user")
		s.SetInt(loginAttemptsKey, 5)

		err := s.Save()
		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"user_name": "test-user"`)
		assert.Contains(t, string(content), `"login_attempts": 5`)
	})
}

func TestService_LoadConfigFile(t *testing.T) {
	tempDir := setupTestDirectory(t)
	configPath := filepath.Join(tempDir, "test-config.json")

	// Create a config file with multiple values
	configContent := `{
		"export_path": "/test/export/path",
		"max_export_size": 5000,
		"last_loaded_vault": "test-vault",
		"custom_key": "custom_value"
	}`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Load the config
	s := NewService(preferences.Options{Path: "test-config", Type: preferences.JSON})

	// Verify all values loaded correctly
	assert.Equal(t, "/test/export/path", s.GetString(preferences.ExportPathKey))
	assert.Equal(t, 5000, s.GetInt(preferences.MaxExportSizeKey))
	assert.Equal(t, "test-vault", s.GetString(preferences.LastLoadedVaultKey))
	assert.Equal(t, "custom_value", s.Get(preferences.Key{ID: "custom_key"}))

	// Modify a value and save
	s.SetString(preferences.ExportPathKey, "/modified/path")
	s.SetInt(preferences.MaxExportSizeKey, 8000)
	require.NoError(t, s.Save())

	// Reload the config in a new service
	s2 := NewService(preferences.Options{Path: "test-config", Type: preferences.JSON})

	// Verify modifications persisted
	assert.Equal(t, "/modified/path", s2.GetString(preferences.ExportPathKey))
	assert.Equal(t, 8000, s2.GetInt(preferences.MaxExportSizeKey))
	assert.Equal(t, "test-vault", s2.GetString(preferences.LastLoadedVaultKey))
	assert.Equal(t, "custom_value", s2.Get(preferences.Key{ID: "custom_key"}))
}

func TestService_Concurrency(t *testing.T) {
	setupTestDirectory(t)
	s := NewService(preferences.Options{Type: preferences.JSON})
	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			key := preferences.Key{ID: fmt.Sprintf("key-%d", n)}
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
	assert.Len(t, allSettings, numGoroutines+len(s.Keys()))
}
