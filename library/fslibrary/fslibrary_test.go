package fslibrary

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	lpng "image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/operation/opcache"
	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/blob/pblob"
	"github.com/digital-foxy/card-client/store/record"
	"github.com/digital-foxy/card-client/store/record/erecord"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/jsonx"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testRegistry = opcache.NewRegistry(opcache.DefaultIdGenerator())

type testRecordBuilder struct{}

func (t testRecordBuilder) Build(path string) (record.Store, error) {
	return erecord.InMemoryStore()
}

func testOptions(tempRoot string) library.Options {
	return library.Options{
		Path:          tempRoot,
		MaxVaults:     5,
		Workers:       4,
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

func TestNewFsLibrary(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(tempRoot string) error
		path          string
		expectedCount int
		expectError   bool
		validateFunc  func(t *testing.T, lib *FsLibrary, tempRoot string)
	}{
		{
			name:          "empty directory",
			setupFunc:     func(tempRoot string) error { return nil },
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "with existing vaults",
			setupFunc: func(tempRoot string) error {
				if err := os.Mkdir(filepath.Join(tempRoot, "vault1"), 0755); err != nil {
					return err
				}
				return os.Mkdir(filepath.Join(tempRoot, "vault2"), 0755)
			},
			expectedCount: 2,
			expectError:   false,
			validateFunc: func(t *testing.T, lib *FsLibrary, tempRoot string) {
				names := lib.Names()
				assert.Contains(t, names, library.VaultName("vault1"))
				assert.Contains(t, names, library.VaultName("vault2"))
			},
		},
		{
			name: "removes non-directories",
			setupFunc: func(tempRoot string) error {
				if err := os.Mkdir(filepath.Join(tempRoot, "vault1"), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(tempRoot, "somefile.txt"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tempRoot, "config.json"), []byte("{}"), 0644)
			},
			expectedCount: 1,
			expectError:   false,
			validateFunc: func(t *testing.T, lib *FsLibrary, tempRoot string) {
				_, err := os.Stat(filepath.Join(tempRoot, "somefile.txt"))
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(filepath.Join(tempRoot, "config.json"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name:        "invalid directory",
			path:        "/nonexistent/path/that/does/not/exist",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempRoot string
			if tt.path != "" {
				tempRoot = tt.path
			} else {
				tempRoot = t.TempDir()
				if tt.setupFunc != nil {
					require.NoError(t, tt.setupFunc(tempRoot))
				}
			}

			lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, lib)
			assert.Equal(t, tt.expectedCount, lib.Count())

			if tt.validateFunc != nil {
				tt.validateFunc(t, lib, tempRoot)
			}
		})
	}
}

// Names and Count tests

func TestNames(t *testing.T) {
	tests := []struct {
		name           string
		vaultsToCreate []string
		expectedNames  []library.VaultName
		expectedCount  int
	}{
		{
			name:           "empty",
			vaultsToCreate: nil,
			expectedNames:  nil,
			expectedCount:  0,
		},
		{
			name:           "returns all vaults",
			vaultsToCreate: []string{"vault1", "vault2", "vault3"},
			expectedNames:  []library.VaultName{"vault1", "vault2", "vault3"},
			expectedCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
			require.NoError(t, err)

			for _, vaultName := range tt.vaultsToCreate {
				require.NoError(t, lib.Create(library.VaultName(vaultName)))
			}

			names := lib.Names()
			if tt.expectedCount == 0 {
				assert.Empty(t, names)
			} else {
				assert.Len(t, names, tt.expectedCount)
				for _, expected := range tt.expectedNames {
					assert.Contains(t, names, expected)
				}
			}
		})
	}
}

func TestCount(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(opts, testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)

	_, err = lib.Load("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoad_CreatesManifest(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
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
	lib, err := NewFsLibrary(opts, testRegistry)
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
	lib, err := NewFsLibrary(opts, testRegistry)
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
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)

	err = lib.Create("vault-with-dashes")
	assert.NoError(t, err)

	err = lib.Create("vault_with_underscores")
	assert.NoError(t, err)
}

// ImportVault tests

func createTestPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := lpng.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func createTestCharacterCard(t *testing.T) *png.CharacterCard {
	rawCard, err := png.FromBytes(createTestPNG()).First().Get()
	require.NoError(t, err)

	card, err := rawCard.Decode()
	require.NoError(t, err)
	return card
}

func createTestRecord(platformID string) resource.Record {
	now := timestamp.NowNano()
	return resource.Record{
		ID: resource.RID(1),
		ImportData: resource.ImportData{
			ImportTime:  now,
			ImportIndex: 0,
		},
		InfoData: resource.InfoData{
			Source:        source.ChubAI,
			NormalizedURL: "https://example.com/" + platformID,
			DirectURL:     "https://example.com/direct/" + platformID,
			PlatformID:    platformID,
			CharacterID:   "char_" + platformID,
			Name:          "Test Name " + platformID,
			Title:         "Test Title " + platformID,
			Tagline:       "Test Tagline",
			CreateTime:    now,
			UpdateTime:    now,
		},
		Creator: resource.Creator{
			ID:         resource.CID("creator_" + platformID),
			Nickname:   "Creator Nickname",
			Username:   "creator_username",
			PlatformID: "creator_123",
			Source:     source.ChubAI,
		},
	}
}

func setupExportFolder(t *testing.T, records map[resource.RID]resource.Record) string {
	exportPath := t.TempDir()

	// Write index.json
	indexPath := filepath.Join(exportPath, "index.json")
	err := jsonx.ToFile(records, indexPath)
	require.NoError(t, err)

	// Create PNG files for each record
	for rid := range records {
		card := createTestCharacterCard(t)
		rawCard, err := card.Encode()
		require.NoError(t, err)
		cardBytes, err := rawCard.ToBytes()
		require.NoError(t, err)

		filename := fmt.Sprintf("%d_TestCard.png", rid)
		cardPath := filepath.Join(exportPath, filename)
		err = os.WriteFile(cardPath, cardBytes, 0644)
		require.NoError(t, err)
	}

	return exportPath
}

func createRecordWithID(platformID string, id resource.RID) resource.Record {
	rec := createTestRecord(platformID)
	rec.ID = id
	return rec
}

func waitForOperation(opID operation.ID) operation.Report {
	for {
		reports := testRegistry.ListReports()
		for _, report := range reports {
			if report.Details.ID == opID && report.Details.Status != operation.Ongoing {
				return report.Report
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestImportVault_Success(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)
	fsLib := lib

	// Create test records
	records := map[resource.RID]resource.Record{
		1: createRecordWithID("card1", 1),
		2: createRecordWithID("card2", 2),
		3: createRecordWithID("card3", 3),
	}

	// Setup export folder
	exportPath := setupExportFolder(t, records)

	// Import the vault
	opID, err := fsLib.ImportVault(exportPath)
	require.NoError(t, err)

	report := waitForOperation(opID)
	assert.Equal(t, 3, report.NoSuccesses)

	// Verify a new vault was created
	assert.Equal(t, 1, lib.Count())
}

func TestImportVault_EmptyIndex(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)
	fsLib := lib

	// Create empty export folder
	records := map[resource.RID]resource.Record{}
	exportPath := setupExportFolder(t, records)

	// Import the vault
	opID, err := fsLib.ImportVault(exportPath)
	require.NoError(t, err)

	report := waitForOperation(opID)
	assert.Equal(t, 0, report.NoSuccesses)
}

func TestImportVault_MissingIndexFile(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)
	fsLib := lib

	// Create export folder without index.json
	exportPath := t.TempDir()

	// Import starts but fails internally (operation still created)
	opID, err := fsLib.ImportVault(exportPath)
	require.NoError(t, err)

	report := waitForOperation(opID)
	assert.Equal(t, 0, report.NoSuccesses)
}

func TestImportVault_MissingCardFile(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)
	fsLib := lib

	// Create test records but don't create PNG files
	records := map[resource.RID]resource.Record{
		1: createRecordWithID("card1", 1),
	}

	exportPath := t.TempDir()
	indexPath := filepath.Join(exportPath, "index.json")
	err = jsonx.ToFile(records, indexPath)
	require.NoError(t, err)

	// Import should succeed but with 0 imported (card file missing)
	opID, err := fsLib.ImportVault(exportPath)
	require.NoError(t, err)

	report := waitForOperation(opID)
	assert.Equal(t, 0, report.NoSuccesses)
}

func TestImportVault_ParallelProcessing(t *testing.T) {
	tempRoot := t.TempDir()
	lib, err := NewFsLibrary(testOptions(tempRoot), testRegistry)
	require.NoError(t, err)
	fsLib := lib

	// Create many test records to test parallel processing
	records := make(map[resource.RID]resource.Record)
	for i := 1; i <= 10; i++ {
		records[resource.RID(i)] = createRecordWithID(fmt.Sprintf("card%d", i), resource.RID(i))
	}

	// Setup export folder
	exportPath := setupExportFolder(t, records)

	// Import the vault
	opID, err := fsLib.ImportVault(exportPath)
	require.NoError(t, err)

	report := waitForOperation(opID)
	assert.Equal(t, 10, report.NoSuccesses)
}
