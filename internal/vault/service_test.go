package vault

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Run("Loads existing valid vaults from disk and cleans invalid ones", func(t *testing.T) {
		tempDir := t.TempDir()
		createValidVaultFS(t, tempDir, "vault1")
		createValidVaultFS(t, tempDir, "vault2")
		require.NoError(t, os.Mkdir(tempDir+"/invalid-vault", 0755))

		options := opts.VaultOptions{RootDir: tempDir}
		s := NewService(options)

		assert.Equal(t, 2, s.VaultCount())
		names := s.GetVaultNames()
		assert.ElementsMatch(t, []string{"vault1", "vault2"}, names)
	})

	t.Run("Applies default vault limit correctly", func(t *testing.T) {
		tempDir := t.TempDir()
		s := NewService(opts.VaultOptions{RootDir: tempDir, VaultLimit: 0})
		assert.Equal(t, defaultVaultLimit, s.vaultLimit)

		s = NewService(opts.VaultOptions{RootDir: tempDir, VaultLimit: -10})
		assert.Equal(t, defaultVaultLimit, s.vaultLimit)
	})

	t.Run("Applies custom vault limit", func(t *testing.T) {
		tempDir := t.TempDir()
		s := NewService(opts.VaultOptions{RootDir: tempDir, VaultLimit: 10})
		assert.Equal(t, 10, s.vaultLimit)
	})
}

func TestService_Getters(t *testing.T) {
	tempDir := t.TempDir()
	createValidVaultFS(t, tempDir, "v1")
	createValidVaultFS(t, tempDir, "v2")
	s := NewService(opts.VaultOptions{RootDir: tempDir})

	assert.Equal(t, 2, s.VaultCount())

	names := s.GetVaultNames()
	assert.ElementsMatch(t, []string{"v1", "v2"}, names)

	vaults := s.GetVaults()
	assert.Len(t, vaults, 2)

	v1, exists1 := s.GetVault("v1")
	assert.True(t, exists1)
	assert.Equal(t, "v1", v1.Name)

	_, exists2 := s.GetVault("non-existent")
	assert.False(t, exists2)
}

func TestService_CreateVault(t *testing.T) {
	t.Run("Successfully creates a new vault", func(t *testing.T) {
		tempDir := t.TempDir()
		s := NewService(opts.VaultOptions{RootDir: tempDir})
		require.Equal(t, 0, s.VaultCount())

		v, err := s.CreateVault("new-vault")
		require.NoError(t, err)

		assert.Equal(t, "new-vault", v.Name)
		assert.Equal(t, 1, s.VaultCount())

		_, exists := s.GetVault("new-vault")
		assert.True(t, exists)

		s2 := NewService(opts.VaultOptions{RootDir: tempDir})
		assert.Equal(t, 1, s2.VaultCount())
	})

	t.Run("Fails if vault already exists in cache", func(t *testing.T) {
		tempDir := t.TempDir()
		s := NewService(opts.VaultOptions{RootDir: tempDir})
		_, _ = s.CreateVault("existing-vault")

		_, err := s.CreateVault("existing-vault")
		require.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrVaultAlreadyExists)
	})

	t.Run("Fails if vault limit is exceeded", func(t *testing.T) {
		tempDir := t.TempDir()
		s := NewService(opts.VaultOptions{RootDir: tempDir, VaultLimit: 1})
		_, _ = s.CreateVault("first-vault")

		_, err := s.CreateVault("second-vault")
		require.Error(t, err)
		assert.ErrorIs(t, err, vault.ErrVaultLimitExceeded)
		assert.Equal(t, 1, s.VaultCount())
	})
}

func TestService_DeleteVault(t *testing.T) {
	t.Run("Successfully deletes an existing vault", func(t *testing.T) {
		tempDir := t.TempDir()
		createValidVaultFS(t, tempDir, "vault-to-delete")
		s := NewService(opts.VaultOptions{RootDir: tempDir})
		require.Equal(t, 1, s.VaultCount())

		err := s.DeleteVault("vault-to-delete")
		require.NoError(t, err)

		assert.Equal(t, 0, s.VaultCount())
		_, exists := s.GetVault("vault-to-delete")
		assert.False(t, exists)

		s2 := NewService(opts.VaultOptions{RootDir: tempDir})
		assert.Equal(t, 0, s2.VaultCount())
	})

	t.Run("Returns nil for non-existent vault", func(t *testing.T) {
		tempDir := t.TempDir()
		s := NewService(opts.VaultOptions{RootDir: tempDir})
		err := s.DeleteVault("non-existent")
		assert.NoError(t, err)
	})
}

func TestService_Concurrency(t *testing.T) {
	tempDir := t.TempDir()
	s := NewService(opts.VaultOptions{RootDir: tempDir, VaultLimit: 20})
	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			vaultName := fmt.Sprintf("concurrent-vault-%d", n)

			_, err := s.CreateVault(vaultName)
			if err == nil {
				s.GetVault(vaultName)
				s.DeleteVault(vaultName)
			}

			s.GetVaults()
			s.VaultCount()
		}(i)
	}

	wg.Wait()

	assert.LessOrEqual(t, s.VaultCount(), 20)
}
