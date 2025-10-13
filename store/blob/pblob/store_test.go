package pblob

import (
	"bytes"
	"context"
	"image"
	"image/color"
	lpng "image/png"
	"sync"
	"testing"

	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
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

func createTestCard(t *testing.T) *png.RawCard {
	rawCard, err := png.FromBytes(createTestPNG()).First().Get()
	require.NoError(t, err)
	return rawCard
}

func newTestStore(t *testing.T, maxVersions int) *Store {
	store, err := New(t.TempDir(), Options{MaxVersions: maxVersions, ThumbnailSize: testThumbnailSize})
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store.(*Store)
}

// Basic CRUD operations

func TestPutGet(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	retrieved, err := store.Get(1, 100)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestGetBytes(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	bytes, err := store.GetBytes(1, 100)
	require.NoError(t, err)
	assert.NotNil(t, bytes)
	assert.NotEmpty(t, bytes)
}

func TestGetBytes_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	_, err := store.GetBytes(1, 100)
	assert.Error(t, err)
}

func TestGet_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	_, err := store.Get(1, 100)
	assert.Error(t, err)
}

func TestThumbnail(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	thumbnail, err := store.Thumbnail(1)
	require.NoError(t, err)
	assert.NotNil(t, thumbnail)

	bounds := thumbnail.Bounds()
	assert.Equal(t, testThumbnailSize, bounds.Dx())
	assert.Equal(t, testThumbnailSize, bounds.Dy())
}

func TestThumbnail_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)
	_, err := store.Thumbnail(1)
	assert.Error(t, err)
}

func TestThumbnail_UpdatesOnPut(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	// Put first version
	require.NoError(t, store.Put(1, 100, card))
	thumbnail1, err := store.Thumbnail(1)
	require.NoError(t, err)

	// Put second version - thumbnail should update
	require.NoError(t, store.Put(1, 200, card))
	thumbnail2, err := store.Thumbnail(1)
	require.NoError(t, err)

	// Both thumbnails should exist and have correct dimensions
	assert.Equal(t, testThumbnailSize, thumbnail1.Bounds().Dx())
	assert.Equal(t, testThumbnailSize, thumbnail2.Bounds().Dx())
}

func TestThumbnail_DeletedWithRID(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))
	_, err := store.Thumbnail(1)
	require.NoError(t, err)

	// Delete all versions of RID 1
	require.NoError(t, store.Delete(1))

	// Thumbnail should also be deleted
	_, err = store.Thumbnail(1)
	assert.Error(t, err)
}

func TestThumbnailBytes_Success(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	bytes, err := store.ThumbnailBytes(1)
	require.NoError(t, err)
	assert.NotNil(t, bytes)
	assert.NotEmpty(t, bytes)
}

func TestThumbnailBytes_NonExistent(t *testing.T) {
	store := newTestStore(t, 10)

	_, err := store.ThumbnailBytes(999)
	assert.Error(t, err)
}

func TestThumbnailBytes_ClonedData(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	// Get bytes twice
	bytes1, err := store.ThumbnailBytes(1)
	require.NoError(t, err)

	bytes2, err := store.ThumbnailBytes(1)
	require.NoError(t, err)

	// Should be equal content
	assert.Equal(t, bytes1, bytes2)

	// But different slices (cloned)
	bytes1[0] = ^bytes1[0]
	assert.NotEqual(t, bytes1[0], bytes2[0])
}

func TestVersions(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	versions := []timestamp.Nano{100, 200, 300}
	for _, v := range versions {
		require.NoError(t, store.Put(1, v, card))
	}

	assert.Equal(t, versions, store.Versions(1))
}

func TestVersions_Empty(t *testing.T) {
	store := newTestStore(t, 10)
	assert.Empty(t, store.Versions(999))
}

func TestVersions_Sorted(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	// Insert in random order
	versions := []timestamp.Nano{300, 100, 500, 200, 400}
	for _, v := range versions {
		require.NoError(t, store.Put(1, v, card))
	}

	result := store.Versions(1)
	assert.Equal(t, []timestamp.Nano{100, 200, 300, 400, 500}, result)
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

	// Version management should NOT work inside transaction
	err := store.WithTx(func(tx blob.TxStore) error {
		require.NoError(t, tx.Put(1, 100, card))
		require.NoError(t, tx.Put(1, 200, card))
		require.NoError(t, tx.Put(1, 300, card))

		versions := tx.Versions(1)
		assert.Equal(t, 3, len(versions))
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []timestamp.Nano{100, 200, 300}, store.Versions(1))
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

func TestWithTx_Commit(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	err := store.WithTx(func(tx blob.TxStore) error {
		return tx.Put(1, 100, card)
	})
	require.NoError(t, err)

	assert.Equal(t, []timestamp.Nano{100}, store.Versions(1))
}

func TestWithTx_Rollback(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	err := store.WithTx(func(tx blob.TxStore) error {
		require.NoError(t, tx.Put(1, 100, card))
		return assert.AnError
	})
	require.Error(t, err)

	assert.Empty(t, store.Versions(1))
}

func TestWithTx_Nested(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

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
}

func TestWithTx_NestedRollback(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

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
}

func TestWithReadTx(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)

	require.NoError(t, store.Put(1, 100, card))

	err := store.WithReadTx(func(tx blob.TxReadStore) error {
		_, err := tx.Get(1, 100)
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
	card := createTestCard(t)

	var wg sync.WaitGroup
	rids := []resource.RID{1, 2, 3, 4, 5}

	// Concurrent writes
	for _, rid := range rids {
		wg.Add(1)
		go func(r resource.RID) {
			defer wg.Done()
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
	retrieved, err := store.Get(rid, 100)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestEdgeCases_NegativeVersion(t *testing.T) {
	store := newTestStore(t, 10)
	card := createTestCard(t)
	negVersion := timestamp.Nano(-100)

	require.NoError(t, store.Put(1, negVersion, card))
	retrieved, err := store.Get(1, negVersion)
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
