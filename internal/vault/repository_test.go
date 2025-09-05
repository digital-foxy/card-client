package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/r3dpixel/card-client/services/vault"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createValidVaultFS(t *testing.T, rootDir, name string) {
	t.Helper()
	v := newVault(rootDir, name)
	err := createPaths(&v)
	require.NoError(t, err)
}

func createInvalidVaultFS(t *testing.T, rootDir, name string) {
	t.Helper()
	vaultDir := filepath.Join(rootDir, name)
	require.NoError(t, os.MkdirAll(vaultDir, 0755))
}

func TestNewRepository(t *testing.T) {
	repo := newRepository("/tmp/my-vaults")
	require.NotNil(t, repo)
	assert.Equal(t, "/tmp/my-vaults", repo.rootDir)
}

func TestRepository_DeleteVault(t *testing.T) {
	t.Run("Deletes an existing vault", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		vaultName := "vault-to-delete"
		createValidVaultFS(t, tempDir, vaultName)

		err := repo.DeleteVault(vaultName)
		require.NoError(t, err)

		vaultPath := filepath.Join(tempDir, vaultName)
		_, err = os.Stat(vaultPath)
		assert.True(t, os.IsNotExist(err), "Vault directory should be gone")
	})

	t.Run("Does not error on non-existent vault", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)

		err := repo.DeleteVault("non-existent-vault")
		assert.NoError(t, err)
	})
}

func TestRepository_All(t *testing.T) {
	t.Run("Finds all valid vaults", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		createValidVaultFS(t, tempDir, "vault1")
		createValidVaultFS(t, tempDir, "vault2")

		allVaults := repo.All()
		assert.Len(t, allVaults, 2)
		assert.Contains(t, allVaults, "vault1")
		assert.Contains(t, allVaults, "vault2")
	})

	t.Run("Cleans up invalid vaults and returns only valid ones", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		createValidVaultFS(t, tempDir, "valid-vault")
		createInvalidVaultFS(t, tempDir, "invalid-vault")

		allVaults := repo.All()
		assert.Len(t, allVaults, 1)
		assert.Contains(t, allVaults, "valid-vault")

		invalidPath := filepath.Join(tempDir, "invalid-vault")
		_, err := os.Stat(invalidPath)
		assert.True(t, os.IsNotExist(err), "Invalid vault should have been cleaned up")
	})

	t.Run("Returns empty map for an empty directory", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		allVaults := repo.All()
		require.NotNil(t, allVaults)
		assert.Empty(t, allVaults)
	})

	t.Run("Returns empty map and logs error on unreadable directory", func(t *testing.T) {
		var logBuffer bytes.Buffer
		originalLogger := log.Logger
		log.Logger = zerolog.New(&logBuffer)
		defer func() { log.Logger = originalLogger }()

		tempDir := t.TempDir()
		require.NoError(t, os.Chmod(tempDir, 0000))
		repo := newRepository(tempDir)

		allVaults := repo.All()
		assert.Empty(t, allVaults)
		assert.Contains(t, logBuffer.String(), "Failed to read directory")

		require.NoError(t, os.Chmod(tempDir, 0755))
	})
}

func TestRepository_CreateVault(t *testing.T) {
	t.Run("Successfully creates a new vault", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		vaultName := "new-vault"

		v, err := repo.CreateVault(vaultName)
		require.NoError(t, err)
		assert.Equal(t, vaultName, v.Name)

		assert.True(t, isValid(&v), "Vault should be valid on the filesystem")
	})

	t.Run("Returns ErrVaultAlreadyExists if vault is valid", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		vaultName := "existing-vault"
		createValidVaultFS(t, tempDir, vaultName)

		_, err := repo.CreateVault(vaultName)
		require.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrVaultAlreadyExists)
	})

	t.Run("Cleans up and creates vault if remnant was invalid", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		vaultName := "partial-vault"
		createInvalidVaultFS(t, tempDir, vaultName)

		v, err := repo.CreateVault(vaultName)
		require.NoError(t, err)
		assert.True(t, isValid(&v), "A valid vault should be created")
	})

	t.Run("Fails to create vault with permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newRepository(tempDir)
		require.NoError(t, os.Chmod(tempDir, 0555)) // Make root dir read-only
		defer func() { require.NoError(t, os.Chmod(tempDir, 0755)) }()

		_, err := repo.CreateVault("no-can-do")
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrPermission)
	})
}
