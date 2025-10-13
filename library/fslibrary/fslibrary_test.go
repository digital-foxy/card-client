package fslibrary

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/blob/pblob"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/record/erecord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRecordBuilder struct{}

func (t testRecordBuilder) Build(path string) (record.Store, error) {
	return erecord.InMemoryStore()
}

func testOptions(tempRoot string) library.Options {
	return library.Options{
		Path:          tempRoot,
		MaxVaults:     5,
		RecordStorage: library.EntSQL,
		BlobStorage:   library.Pebble,
		RecordStorages: map[library.RecordStorage]record.Builder{
			library.EntSQL: testRecordBuilder{},
		},
		BlobStorages: map[library.BlobStorage]blob.Builder{
			library.Pebble: pblob.Builder{MaxVersions: 5, ThumbnailSize: 256},
		},
	}
}

// NewFsLibrary tests

func TestNewFsLibrary_EmptyDirectory(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))

	require.NoError(t, err)
	assert.NotNil(t, lib)
	assert.Equal(t, 0, lib.Count())
}

func TestNewFsLibrary_WithExistingVaults(t *testing.T) {
	tempRoot := t.TempDir()

	// Create existing vault directories
	require.NoError(t, os.Mkdir(filepath.Join(tempRoot, "vault1"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tempRoot, "vault2"), 0755))

	lib, err := NewFsLibrary(testOptions(tempRoot))

	require.NoError(t, err)
	assert.Equal(t, 2, lib.Count())
	names := lib.Names()
	assert.Contains(t, names, library.VaultName("vault1"))
	assert.Contains(t, names, library.VaultName("vault2"))
}

func TestNewFsLibrary_RemovesNonDirectories(t *testing.T) {
	tempRoot := t.TempDir()

	// Create files and directories
	require.NoError(t, os.Mkdir(filepath.Join(tempRoot, "vault1"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempRoot, "somefile.txt"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempRoot, "config.json"), []byte("{}"), 0644))

	lib, err := NewFsLibrary(testOptions(tempRoot))

	require.NoError(t, err)
	assert.Equal(t, 1, lib.Count()) // Only vault1 should be loaded

	// Files should be removed
	_, err = os.Stat(filepath.Join(tempRoot, "somefile.txt"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(tempRoot, "config.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestNewFsLibrary_InvalidDirectory(t *testing.T) {
	_, err := NewFsLibrary(testOptions("/nonexistent/path/that/does/not/exist"))
	assert.Error(t, err)
}

// Names() tests

func TestNames_Empty(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	names := lib.Names()
	assert.Empty(t, names)
}

func TestNames_ReturnsAllVaults(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	require.NoError(t, lib.Create("vault1"))
	require.NoError(t, lib.Create("vault2"))
	require.NoError(t, lib.Create("vault3"))

	names := lib.Names()
	assert.Len(t, names, 3)
	assert.Contains(t, names, library.VaultName("vault1"))
	assert.Contains(t, names, library.VaultName("vault2"))
	assert.Contains(t, names, library.VaultName("vault3"))
}

// Count() tests

func TestCount_Empty(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	assert.Equal(t, 0, lib.Count())
}

func TestCount_AfterOperations(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	assert.Equal(t, 0, lib.Count())

	require.NoError(t, lib.Create("vault1"))
	assert.Equal(t, 1, lib.Count())

	require.NoError(t, lib.Create("vault2"))
	assert.Equal(t, 2, lib.Count())

	require.NoError(t, lib.Delete("vault1"))
	assert.Equal(t, 1, lib.Count())
}

// Create() tests

func TestCreate_Success(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	err = lib.Create("test-vault")
	require.NoError(t, err)

	// Verify vault directory exists
	vaultPath := filepath.Join(tempRoot, "test-vault")
	info, err := os.Stat(vaultPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify vault is in library
	names := lib.Names()
	assert.Contains(t, names, library.VaultName("test-vault"))
}

func TestCreate_DuplicateName(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	require.NoError(t, lib.Create("vault1"))

	err = lib.Create("vault1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreate_MaxVaultsReached(t *testing.T) {
	tempRoot := t.TempDir()
	opts := testOptions(tempRoot)
	opts.MaxVaults = 2
	lib, err := NewFsLibrary(opts)
	require.NoError(t, err)

	require.NoError(t, lib.Create("vault1"))
	require.NoError(t, lib.Create("vault2"))

	err = lib.Create("vault3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum vaults")
}

// Delete() tests

func TestDelete_Success(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	require.NoError(t, lib.Create("test-vault"))

	err = lib.Delete("test-vault")
	require.NoError(t, err)

	// Verify vault directory removed from disk
	vaultPath := filepath.Join(tempRoot, "test-vault")
	_, err = os.Stat(vaultPath)
	assert.True(t, os.IsNotExist(err))
}

func TestDelete_AllowsRecreation(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	require.NoError(t, lib.Create("vault1"))
	require.NoError(t, lib.Delete("vault1"))

	// Should be able to create again with same name
	err = lib.Create("vault1")
	assert.NoError(t, err)
}

// Load() tests

func TestLoad_Success(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	require.NoError(t, lib.Create("test-vault"))

	vault, err := lib.Load("test-vault")
	require.NoError(t, err)
	assert.NotNil(t, vault.Catalog)
	assert.Equal(t, library.VaultName("test-vault"), vault.Name)

	// Cleanup
	require.NoError(t, vault.Catalog.Close())
}

func TestLoad_NonExistentVault(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	_, err = lib.Load("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoad_CreatesManifest(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	require.NoError(t, lib.Create("test-vault"))

	vault, err := lib.Load("test-vault")
	require.NoError(t, err)
	t.Cleanup(func() { vault.Catalog.Close() })

	// Verify manifest file exists
	manifestPath := filepath.Join(tempRoot, "test-vault", library.ManifestPath)
	_, err = os.Stat(manifestPath)
	assert.NoError(t, err)
}

func TestConcurrentCreate(t *testing.T) {
	tempRoot := t.TempDir()
	opts := testOptions(tempRoot)
	opts.MaxVaults = 100
	lib, err := NewFsLibrary(opts)
	require.NoError(t, err)

	// Create vaults concurrently
	const numVaults = 10
	errors := make(chan error, numVaults)

	for i := 0; i < numVaults; i++ {
		go func(n int) {
			vaultName := library.VaultName(fmt.Sprintf("vault-%d", n))
			errors <- lib.Create(vaultName)
		}(i)
	}

	for i := 0; i < numVaults; i++ {
		err := <-errors
		assert.NoError(t, err)
	}

	assert.Equal(t, numVaults, lib.Count())
}

func TestConcurrentReadWrite(t *testing.T) {
	tempRoot := t.TempDir()
	opts := testOptions(tempRoot)
	opts.MaxVaults = 100
	lib, err := NewFsLibrary(opts)
	require.NoError(t, err)

	require.NoError(t, lib.Create("vault1"))
	require.NoError(t, lib.Create("vault2"))

	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func() {
			_ = lib.Names()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		go func(n int) {
			vaultName := library.VaultName(fmt.Sprintf("vault-%d", n+3))
			_ = lib.Create(vaultName)
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	assert.True(t, lib.Count() >= 2)
}

func TestCreate_SpecialCharacters(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot))
	require.NoError(t, err)

	err = lib.Create("vault-with-dashes")
	assert.NoError(t, err)

	err = lib.Create("vault_with_underscores")
	assert.NoError(t, err)
}
