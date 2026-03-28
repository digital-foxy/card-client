package pblob

import (
	"bytes"
	"context"
	"image"
	"image/color"
	lpng "image/png"
	"sync"
	"testing"

	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions
const testThumbnailSize = 256

func createTestPNG() []byte {
	// Create a 512x512 image with gradient pattern
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

func createTestCard(t *testing.T) *png.CharacterCard {
	rawCard, err := png.FromBytes(createTestPNG()).First().Get()
	require.NoError(t, err)

	card, err := rawCard.Decode()
	require.NoError(t, err)
	return card
}

func newTestStore(t *testing.T, maxVersions int) *Store {
	store, err := New(t.TempDir(), Options{MaxVersions: maxVersions, ThumbnailSize: testThumbnailSize})
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store.(*Store)
}

// Basic CRUD operations

func TestPutGetRawCard(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	retrieved, err := store.GetRawCard(1, 100)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Also test GetCharacterCard
	characterCard, err := store.GetCharacterCard(1, 100)
	require.NoError(t, err)
	assert.NotNil(t, characterCard)
	assert.NotNil(t, characterCard.Sheet)
}

func TestGetBytes(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*testing.T, *Store) *png.CharacterCard
		expectError  bool
		validateFunc func(*testing.T, *Store, []byte)
	}{
		{
			name: "GetRawCardBytes success",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				bytes, err := store.GetRawCardBytes(1, 100)
				require.NoError(t, err)
				assert.NotNil(t, bytes)
				assert.NotEmpty(t, bytes)

				// Verify it's PNG bytes by parsing
				_, err = png.FromBytes(bytes).First().Get()
				require.NoError(t, err)
			},
		},
		{
			name:        "GetRawCardBytes non-existent",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return nil },
			expectError: true,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				_, err := store.GetRawCardBytes(1, 100)
				assert.Error(t, err)
			},
		},
		{
			name: "GetRawCardBytes cloned data",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				bytes1, err := store.GetRawCardBytes(1, 100)
				require.NoError(t, err)

				bytes2, err := store.GetRawCardBytes(1, 100)
				require.NoError(t, err)

				// Should be equal content
				assert.Equal(t, bytes1, bytes2)

				// But different slices (cloned)
				bytes1[0] = ^bytes1[0]
				assert.NotEqual(t, bytes1[0], bytes2[0])
			},
		},
		{
			name: "GetSheetBytes success",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				bytes, err := store.GetSheetBytes(1, 100)
				require.NoError(t, err)
				assert.NotNil(t, bytes)
				assert.NotEmpty(t, bytes)

				// Verify sheet can be retrieved and matches
				sheet1, err := store.GetSheet(1, 100)
				require.NoError(t, err)

				sheet2, err := character.FromBytes(bytes)
				require.NoError(t, err)

				assert.Equal(t, sheet1, sheet2)
			},
		},
		{
			name:        "GetSheetBytes non-existent",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return nil },
			expectError: true,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				_, err := store.GetSheetBytes(1, 100)
				assert.Error(t, err)
			},
		},
		{
			name: "GetSheetBytes cloned data",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				bytes1, err := store.GetSheetBytes(1, 100)
				require.NoError(t, err)

				bytes2, err := store.GetSheetBytes(1, 100)
				require.NoError(t, err)

				// Should be equal content
				assert.Equal(t, bytes1, bytes2)

				// But different slices (cloned)
				bytes1[0] = ^bytes1[0]
				assert.NotEqual(t, bytes1[0], bytes2[0])
			},
		},
		{
			name: "ThumbnailBytes success",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				bytes, err := store.ThumbnailBytes(1)
				require.NoError(t, err)
				assert.NotNil(t, bytes)
				assert.NotEmpty(t, bytes)
			},
		},
		{
			name:        "ThumbnailBytes non-existent",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return nil },
			expectError: true,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				_, err := store.ThumbnailBytes(999)
				assert.Error(t, err)
			},
		},
		{
			name: "ThumbnailBytes cloned data",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, bytes []byte) {
				bytes1, err := store.ThumbnailBytes(1)
				require.NoError(t, err)

				bytes2, err := store.ThumbnailBytes(1)
				require.NoError(t, err)

				// Should be equal content
				assert.Equal(t, bytes1, bytes2)

				// But different slices (cloned)
				bytes1[0] = ^bytes1[0]
				assert.NotEqual(t, bytes1[0], bytes2[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, 10)
			tt.setupFunc(t, store)
			tt.validateFunc(t, store, nil)
		})
	}
}

func TestGetSheet(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	sheet, err := store.GetSheet(1, 100)
	require.NoError(t, err)
	assert.NotNil(t, sheet)
}

func TestGetSheet_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	_, err := store.GetSheet(1, 100)
	assert.Error(t, err)
}

func TestGetRawCard_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	_, err := store.GetRawCard(1, 100)
	assert.Error(t, err)
}

func TestThumbnail(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*testing.T, *Store) *png.CharacterCard
		expectError  bool
		validateFunc func(*testing.T, *Store)
	}{
		{
			name: "success",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store) {
				thumbnail, err := store.Thumbnail(1)
				require.NoError(t, err)
				assert.NotNil(t, thumbnail)

				bounds := thumbnail.Bounds()
				assert.Equal(t, testThumbnailSize, bounds.Dx())
				assert.Equal(t, testThumbnailSize, bounds.Dy())
			},
		},
		{
			name:        "non-existent",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return nil },
			expectError: true,
			validateFunc: func(t *testing.T, store *Store) {
				_, err := store.Thumbnail(1)
				assert.Error(t, err)
			},
		},
		{
			name: "updates on put",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store) {
				thumbnail1, err := store.Thumbnail(1)
				require.NoError(t, err)

				// Put second version - thumbnail should update
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 200, card))
				thumbnail2, err := store.Thumbnail(1)
				require.NoError(t, err)

				// Both thumbnails should exist and have correct dimensions
				assert.Equal(t, testThumbnailSize, thumbnail1.Bounds().Dx())
				assert.Equal(t, testThumbnailSize, thumbnail2.Bounds().Dx())
			},
		},
		{
			name: "deleted with RID",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				require.NoError(t, store.Put(1, 100, card))
				_, err := store.Thumbnail(1)
				require.NoError(t, err)
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store) {
				// Delete all versions of RID 1
				require.NoError(t, store.Delete(1))

				// Thumbnail should also be deleted
				_, err := store.Thumbnail(1)
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, 10)
			tt.setupFunc(t, store)
			tt.validateFunc(t, store)
		})
	}
}

func TestVersions(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*testing.T, *Store) *png.CharacterCard
		expectError  bool
		validateFunc func(*testing.T, *Store)
	}{
		{
			name: "multiple versions",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				versions := []timestamp.Nano{100, 200, 300}
				for _, v := range versions {
					require.NoError(t, store.Put(1, v, card))
				}
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store) {
				versions := []timestamp.Nano{100, 200, 300}
				assert.Equal(t, versions, store.Versions(1))
			},
		},
		{
			name:        "empty",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return nil },
			expectError: false,
			validateFunc: func(t *testing.T, store *Store) {
				assert.Empty(t, store.Versions(999))
			},
		},
		{
			name: "sorted",
			setupFunc: func(t *testing.T, store *Store) *png.CharacterCard {
				card := createTestCard(t)
				// Insert in random order
				versions := []timestamp.Nano{300, 100, 500, 200, 400}
				for _, v := range versions {
					require.NoError(t, store.Put(1, v, card))
				}
				return card
			},
			expectError: false,
			validateFunc: func(t *testing.T, store *Store) {
				result := store.Versions(1)
				assert.Equal(t, []timestamp.Nano{100, 200, 300, 400, 500}, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, 10)
			tt.setupFunc(t, store)
			tt.validateFunc(t, store)
		})
	}
}

func TestVersionExists(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	exists, err := store.VersionExists(1, 100)
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, store.Put(1, 100, card))

	exists, err = store.VersionExists(1, 100)
	require.NoError(t, err)
	assert.True(t, exists)
}

// Version management

func TestVersionManagement_NewVersions(t *testing.T) {
	store := newTestStore(t, 3)
	card := createTestCard(t)

	for i := 1; i <= 5; i++ {
		require.NoError(t, store.Put(1, timestamp.Nano(i*100), card))
	}

	versions := store.Versions(1)
	assert.Equal(t, 3, len(versions))
	assert.Equal(t, []timestamp.Nano{300, 400, 500}, versions)
}

func TestVersionManagement_Overwrite(t *testing.T) {
	store := newTestStore(t, 2)
	card := createTestCard(t)

	// Add 2 versions
	require.NoError(t, store.Put(1, 100, card))
	require.NoError(t, store.Put(1, 200, card))
	assert.Equal(t, 2, len(store.Versions(1)))

	// Overwrite version 100 - should NOT delete anything
	require.NoError(t, store.Put(1, 100, card))
	assert.Equal(t, []timestamp.Nano{100, 200}, store.Versions(1))
}

func TestVersionManagement_Zero(t *testing.T) {
	store := newTestStore(t, 0)
	card := createTestCard(t)

	// MaxVersions=0 -> default max versions
	require.NoError(t, store.Put(1, 100, card))
	require.NoError(t, store.Put(1, 200, card))

	versions := store.Versions(1)
	assert.Equal(t, 2, len(versions))
	assert.Equal(t, []timestamp.Nano{100, 200}, versions)
}

func TestVersionManagement_InTransaction(t *testing.T) {
	store := newTestStore(t, 2)
	card := createTestCard(t)

	// Version management works inside transaction
	err := store.WithTx(func(tx blob.TxStore) error {
		require.NoError(t, tx.Put(1, 100, card))
		require.NoError(t, tx.Put(1, 200, card))
		require.NoError(t, tx.Put(1, 300, card))

		versions := tx.Versions(1)
		assert.Equal(t, 2, len(versions))
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []timestamp.Nano{200, 300}, store.Versions(1))
}

// Delete operations

func TestDeleteVersion(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))
	require.NoError(t, store.Put(1, 200, card))
	require.NoError(t, store.DeleteVersion(1, 100))

	assert.Equal(t, []timestamp.Nano{200}, store.Versions(1))
}

func TestDeleteVersion_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	assert.NoError(t, store.DeleteVersion(1, 999))
}

func TestDeleteVersion_BothPNGAndJSON(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	// Verify both PNG and JSON exist
	_, err := store.GetRawCardBytes(1, 100)
	require.NoError(t, err)
	_, err = store.GetSheetBytes(1, 100)
	require.NoError(t, err)

	// Delete version
	require.NoError(t, store.DeleteVersion(1, 100))

	// Both should be deleted
	_, err = store.GetRawCardBytes(1, 100)
	assert.Error(t, err)
	_, err = store.GetSheetBytes(1, 100)
	assert.Error(t, err)
}

func TestDeleteVersions(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	// Add versions 100-500
	for i := 1; i <= 5; i++ {
		require.NoError(t, store.Put(1, timestamp.Nano(i*100), card))
	}

	// Delete range [0, 300) - deletes 100, 200
	require.NoError(t, store.DeleteVersions(1, 0, 300))

	assert.Equal(t, []timestamp.Nano{300, 400, 500}, store.Versions(1))
}

func TestDelete(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	// Add multiple RIDs
	require.NoError(t, store.Put(1, 100, card))
	require.NoError(t, store.Put(1, 200, card))
	require.NoError(t, store.Put(2, 100, card))

	// Delete all of RID 1
	require.NoError(t, store.Delete(1))

	assert.Empty(t, store.Versions(1))
	assert.Equal(t, []timestamp.Nano{100}, store.Versions(2))
}

func TestDelete_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	assert.NoError(t, store.Delete(999))
}

// Transaction tests

func TestWithTx(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*testing.T, *Store) *png.CharacterCard
		expectError  bool
		validateFunc func(*testing.T, *Store, *png.CharacterCard)
	}{
		{
			name:        "commit",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return createTestCard(t) },
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, card *png.CharacterCard) {
				err := store.WithTx(func(tx blob.TxStore) error {
					return tx.Put(1, 100, card)
				})
				require.NoError(t, err)

				assert.Equal(t, []timestamp.Nano{100}, store.Versions(1))
			},
		},
		{
			name:        "rollback",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return createTestCard(t) },
			expectError: true,
			validateFunc: func(t *testing.T, store *Store, card *png.CharacterCard) {
				err := store.WithTx(func(tx blob.TxStore) error {
					require.NoError(t, tx.Put(1, 100, card))
					return assert.AnError
				})
				require.Error(t, err)

				assert.Empty(t, store.Versions(1))
			},
		},
		{
			name:        "nested",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return createTestCard(t) },
			expectError: false,
			validateFunc: func(t *testing.T, store *Store, card *png.CharacterCard) {
				err := store.WithTx(func(tx1 blob.TxStore) error {
					require.NoError(t, tx1.Put(1, 100, card))

					// Nested tx should piggyback
					txStore := tx1.(*Store)
					return txStore.WithTx(func(tx2 blob.TxStore) error {
						return tx2.Put(1, 200, card)
					})
				})
				require.NoError(t, err)

				assert.Equal(t, []timestamp.Nano{100, 200}, store.Versions(1))
			},
		},
		{
			name:        "nested rollback",
			setupFunc:   func(t *testing.T, store *Store) *png.CharacterCard { return createTestCard(t) },
			expectError: true,
			validateFunc: func(t *testing.T, store *Store, card *png.CharacterCard) {
				err := store.WithTx(func(tx1 blob.TxStore) error {
					require.NoError(t, tx1.Put(1, 100, card))

					// Nested tx error causes parent rollback
					txStore := tx1.(*Store)
					return txStore.WithTx(func(tx2 blob.TxStore) error {
						tx2.Put(1, 200, card)
						return assert.AnError
					})
				})
				require.Error(t, err)

				assert.Empty(t, store.Versions(1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, 10)
			card := tt.setupFunc(t, store)
			tt.validateFunc(t, store, card)
		})
	}
}

func TestWithReadTx(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	err := store.WithReadTx(func(tx blob.TxReadStore) error {
		_, err := tx.GetRawCard(1, 100)
		return err
	})
	require.NoError(t, err)
}

func TestWithReadTx_Isolation(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	err := store.WithReadTx(func(tx blob.TxReadStore) error {
		versions := tx.Versions(1)
		assert.Equal(t, []timestamp.Nano{100}, versions)
		return nil
	})
	require.NoError(t, err)
}

func TestWithWriteTx(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	err := store.WithWriteTx(func(tx blob.TxWriteStore) error {
		return tx.DeleteVersion(1, 100)
	})
	require.NoError(t, err)

	assert.Empty(t, store.Versions(1))
}

// Context tests

func TestWithContext(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	ctx := context.Background()
	scoped := store.WithContext(ctx).(*Store)

	require.NoError(t, scoped.Put(1, 100, card))
	assert.Equal(t, []timestamp.Nano{100}, store.Versions(1))
}

func TestWithContext_PreservesTransaction(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	err := store.WithTx(func(tx blob.TxStore) error {
		// WithContext on tx should preserve isTransaction flag
		ctx := context.Background()
		txStore := tx.(*Store)
		scoped := txStore.WithContext(ctx).(*Store)

		return scoped.Put(1, 100, card)
	})
	require.NoError(t, err)

	assert.Equal(t, []timestamp.Nano{100}, store.Versions(1))
}

// Concurrency tests

func TestConcurrentReadWrite(t *testing.T) {
	store := newTestStore(t, 10)

	var wg sync.WaitGroup
	rids := []resource.RID{1, 2, 3, 4, 5}

	// Concurrent writes
	for _, rid := range rids {
		wg.Add(1)
		go func(r resource.RID) {
			defer wg.Done()
			card := createTestCard(t) // Create card per goroutine to avoid race
			for i := 1; i <= 10; i++ {
				store.Put(r, timestamp.Nano(i*100), card)
			}
		}(rid)
	}

	// Concurrent reads
	for _, rid := range rids {
		wg.Add(1)
		go func(r resource.RID) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				store.Versions(r)
			}
		}(rid)
	}

	wg.Wait()

	// Verify all writes succeeded
	for _, rid := range rids {
		versions := store.Versions(rid)
		assert.True(t, len(versions) > 0)
	}
}

// Edge cases

func TestEdgeCases_MaxRID(t *testing.T) {
	store := newTestStore(t, 10)
	rid := resource.RID(^uint64(0))
	card := createTestCard(t)

	require.NoError(t, store.Put(rid, 100, card))
	retrieved, err := store.GetRawCard(rid, 100)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestEdgeCases_NegativeVersion(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)
	negVersion := timestamp.Nano(-100)

	require.NoError(t, store.Put(1, negVersion, card))
	retrieved, err := store.GetRawCard(1, negVersion)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestEdgeCases_AdjacentRIDs(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	// Use adjacent RIDs
	require.NoError(t, store.Put(100, 1, card))
	require.NoError(t, store.Put(101, 1, card))

	// Delete one
	require.NoError(t, store.Delete(100))

	// Other should remain
	assert.Empty(t, store.Versions(100))
	assert.Equal(t, []timestamp.Nano{1}, store.Versions(101))
}

// RIDs tests

func TestRIDs(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	// Empty store
	rids, err := store.RIDs()
	require.NoError(t, err)
	assert.Empty(t, rids)

	// Single RID
	require.NoError(t, store.Put(1, 100, card))
	rids, err = store.RIDs()
	require.NoError(t, err)
	assert.Equal(t, []resource.RID{1}, rids)

	// Multiple RIDs in sorted order
	require.NoError(t, store.Put(5, 100, card))
	require.NoError(t, store.Put(3, 100, card))
	require.NoError(t, store.Put(10, 100, card))
	rids, err = store.RIDs()
	require.NoError(t, err)
	assert.Equal(t, []resource.RID{1, 3, 5, 10}, rids)

	// Multiple versions - RID appears only once
	require.NoError(t, store.Put(1, 200, card))
	require.NoError(t, store.Put(1, 300, card))
	rids, err = store.RIDs()
	require.NoError(t, err)
	assert.Equal(t, []resource.RID{1, 3, 5, 10}, rids)

	// After delete
	require.NoError(t, store.Delete(5))
	rids, err = store.RIDs()
	require.NoError(t, err)
	assert.Equal(t, []resource.RID{1, 3, 10}, rids)

	// After version delete (RID remains)
	require.NoError(t, store.DeleteVersion(1, 100))
	rids, err = store.RIDs()
	require.NoError(t, err)
	assert.Equal(t, []resource.RID{1, 3, 10}, rids)

	// Large RID values
	largeRID := resource.RID(1<<63 - 1)
	require.NoError(t, store.Put(largeRID, 100, card))
	rids, err = store.RIDs()
	require.NoError(t, err)
	assert.Contains(t, rids, largeRID)
	assert.Equal(t, 4, len(rids))
}

func TestRIDs_InTransaction(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	err := store.WithTx(func(tx blob.TxStore) error {
		require.NoError(t, tx.Put(1, 100, card))
		require.NoError(t, tx.Put(2, 100, card))

		rids, err := tx.RIDs()
		assert.NoError(t, err)
		assert.Equal(t, []resource.RID{1, 2}, rids)
		return nil
	})
	require.NoError(t, err)

	rids, err := store.RIDs()
	require.NoError(t, err)
	assert.Equal(t, []resource.RID{1, 2}, rids)
}
