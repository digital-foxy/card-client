package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVault(t *testing.T) {
	rootDir := "/tmp/vaults"
	vaultName := "my-awesome-vault"

	v := newVault(rootDir, vaultName)

	assert.Equal(t, vaultName, v.Name)
	assert.Equal(t, filepath.Join(rootDir, vaultName), v.VaultDir)
	assert.Equal(t, filepath.Join(rootDir, vaultName, cardsDirName), v.CardsDir)
	assert.Equal(t, filepath.Join(rootDir, vaultName, dbFileName), v.DbFilePath)
	assert.Equal(t, filepath.Join(rootDir, vaultName, confFileName), v.ConfFilePath)
}

func TestCreatePaths(t *testing.T) {
	t.Run("Successful creation", func(t *testing.T) {
		tempDir := t.TempDir()
		v := newVault(tempDir, "test-vault")

		err := createPaths(&v)
		require.NoError(t, err)

		_, err = os.Stat(v.CardsDir)
		assert.NoError(t, err, "Cards directory should exist")

		_, err = os.Stat(v.ConfFilePath)
		assert.NoError(t, err, "Config file should exist")
	})

	t.Run("Permission denied", func(t *testing.T) {
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "read-only")
		require.NoError(t, os.Mkdir(readOnlyDir, 0555)) // Read and execute only

		v := newVault(readOnlyDir, "test-vault")

		err := createPaths(&v)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrPermission)
	})
}

func TestIsValid(t *testing.T) {
	t.Run("Vault is valid", func(t *testing.T) {
		tempDir := t.TempDir()
		v := newVault(tempDir, "valid-vault")

		require.NoError(t, createPaths(&v))
		assert.True(t, isValid(&v))
	})

	t.Run("Vault is missing cards directory", func(t *testing.T) {
		tempDir := t.TempDir()
		v := newVault(tempDir, "missing-dir-vault")

		require.NoError(t, os.MkdirAll(v.VaultDir, 0755))
		_, err := os.Create(v.ConfFilePath)
		require.NoError(t, err)

		assert.False(t, isValid(&v))
	})

	t.Run("Vault is missing config file", func(t *testing.T) {
		tempDir := t.TempDir()
		v := newVault(tempDir, "missing-file-vault")

		require.NoError(t, os.MkdirAll(v.CardsDir, 0755))

		assert.False(t, isValid(&v))
	})

	t.Run("Vault is missing both", func(t *testing.T) {
		tempDir := t.TempDir()
		v := newVault(tempDir, "empty-vault")

		require.NoError(t, os.MkdirAll(v.VaultDir, 0755))

		assert.False(t, isValid(&v))
	})
}
