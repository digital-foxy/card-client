package entrecord

import (
	"context"
	"testing"
	"time"

	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStore(t *testing.T) record.Store {
	store, err := NewStore(InMemeryOpts(3))
	require.NoError(t, err, "NewStore failed")
	return store
}

func createTestMetadata(platformID string) *models.Metadata {
	now := timestamp.Now[timestamp.Nano]()
	return &models.Metadata{
		Source: source.ChubAI,
		CardInfo: models.CardInfo{
			NormalizedURL: "https://example.com/" + platformID,
			DirectURL:     "https://example.com/direct/" + platformID,
			PlatformID:    platformID,
			CharacterID:   "char_" + platformID,
			Name:          "Test Name " + platformID,
			Title:         "Test Title " + platformID,
			Tagline:       "Test Tagline " + platformID,
			CreateTime:    now,
			UpdateTime:    now,
			Tags: []models.Tag{
				{Slug: "tag1", Name: "Tag 1"},
				{Slug: "tag2", Name: "Tag 2"},
			},
		},
		CreatorInfo: models.CreatorInfo{
			Nickname:   "Creator Nickname",
			Username:   "creator_username",
			PlatformID: "creator_123",
		},
		BookUpdateTime: now,
	}
}

func TestNewStore(t *testing.T) {
	store, err := NewStore(InMemeryOpts(3))
	require.NoError(t, err)
	defer store.Close()
}

func TestCount(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Test Count with an empty store
	count := store.Count(resource.Filter{})
	assert.Equal(t, 0, count)

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Test Count with one record
	count = store.Count(resource.Filter{})
	assert.Equal(t, 1, count)
}

func TestFindPagedRIDs(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert multiple records
	for i := 1; i <= 5; i++ {
		metadata := createTestMetadata(string(rune('0' + i)))
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, store.InsertRecord(metadata, importData))
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Test pagination
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 3)
	assert.Len(t, rids, 3)

	// Test offset
	rids = store.FindPagedRIDs(resource.Filter{}, 3, 3)
	assert.Len(t, rids, 2)

	// Test empty result
	rids = store.FindPagedRIDs(resource.Filter{}, 10, 3)
	assert.Empty(t, rids)
}

func TestFindRecords(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Find the record
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)
	box := store.FindRecords(rids)

	require.Len(t, box.Items, 1)

	rec := box.Items[0]
	assert.Equal(t, metadata.Name, rec.Name)
	assert.Equal(t, metadata.Title, rec.Title)
	assert.Equal(t, metadata.CardInfo.PlatformID, rec.InfoData.PlatformID)
	assert.Len(t, rec.Tags, 2)
}

func TestFindExportHeaders(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Get RID
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)

	// Update export data
	exportData := resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: timestamp.Now[timestamp.Nano](),
	}
	require.NoError(t, store.UpdateExportData(rids[0], exportData))

	// Find export headers
	box := store.FindExportHeaders(rids)
	require.Len(t, box.Items, 1)

	header := box.Items[0]
	assert.Equal(t, exportData.ExportTime, header.ExportTime)
}

func TestFindURLs(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert records with different URLs
	urls := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/3",
	}

	for i, url := range urls {
		metadata := createTestMetadata(string(rune('1' + i)))
		metadata.NormalizedURL = url
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i + 1,
		}
		require.NoError(t, store.InsertRecord(metadata, importData))
	}

	// Find existing URLs
	searchURLs := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/nonexistent",
	}
	foundURLs := store.FindURLs(searchURLs)

	assert.Len(t, foundURLs, 2)
}

func TestInsertRecord(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	metadata := createTestMetadata("insert_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	require.NoError(t, store.InsertRecord(metadata, importData))

	// Verify insertion
	count := store.Count(resource.Filter{})
	assert.Equal(t, 1, count)

	// Verify tags were inserted
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)
	box := store.FindRecords(rids)
	require.Len(t, box.Items, 1)
	assert.Len(t, box.Items[0].Tags, 2)
}

func TestUpdateRecord(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert a record with tags: tag1, tag2
	metadata := createTestMetadata("update_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Get RID
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NotEmpty(t, rids)

	// Update the record with tags: tag2 (existing), tag3 (new), tag4 (new)
	metadata.Name = "Updated Name"
	metadata.Title = "Updated Title"
	metadata.Tags = []models.Tag{
		{Slug: "tag2", Name: "Tag 2"}, // Existing tag
		{Slug: "tag3", Name: "Tag 3"}, // New tag
		{Slug: "tag4", Name: "Tag 4"}, // New tag
	}
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}

	require.NoError(t, store.UpdateRecord(rids[0], metadata, syncData))

	// Verify update
	box := store.FindRecords([]resource.RID{rids[0]})
	require.Len(t, box.Items, 1)

	rec := box.Items[0]
	assert.Equal(t, "Updated Name", rec.Name)
	assert.Equal(t, "Updated Title", rec.Title)
	assert.Len(t, rec.Tags, 3)

	// Verify tag slugs
	tagSlugs := make([]string, len(rec.Tags))
	for i, tag := range rec.Tags {
		tagSlugs[i] = string(tag.ID)
	}
	assert.Contains(t, tagSlugs, "tag2")
	assert.Contains(t, tagSlugs, "tag3")
	assert.Contains(t, tagSlugs, "tag4")
}

func TestUpdateSyncData(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert a record
	metadata := createTestMetadata("sync_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Get RID
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)

	// Update sync data
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncFailed,
	}
	require.NoError(t, store.UpdateSyncData(rids[0], syncData))

	// Verify update
	box := store.FindRecords([]resource.RID{rids[0]})
	require.Len(t, box.Items, 1)

	assert.Equal(t, resource.SyncFailed, box.Items[0].SyncStatus)
}

func TestUpdateExportData(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert a record
	metadata := createTestMetadata("export_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Get RID
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)

	// Update export data
	exportData := resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: timestamp.Now[timestamp.Nano](),
	}
	require.NoError(t, store.UpdateExportData(rids[0], exportData))

	// Verify update
	box := store.FindExportHeaders([]resource.RID{rids[0]})
	require.Len(t, box.Items, 1)

	assert.Equal(t, exportData.ExportTime, box.Items[0].ExportTime)
}

func TestUpdateFavoriteData(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert multiple records
	for i := 0; i < 3; i++ {
		metadata := createTestMetadata(string(rune('1' + i)))
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i + 1,
		}
		require.NoError(t, store.InsertRecord(metadata, importData))
	}

	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)

	// Update favorite data
	require.NoError(t, store.UpdateFavoriteData(rids, true))

	// Verify update
	box := store.FindRecords(rids)
	for _, rec := range box.Items {
		assert.True(t, rec.Favorite)
	}

	// Unfavorite
	require.NoError(t, store.UpdateFavoriteData(rids[:2], false))

	box = store.FindRecords(rids)
	assert.False(t, box.Items[0].Favorite)
	assert.False(t, box.Items[1].Favorite)
	assert.True(t, box.Items[2].Favorite)
}

func TestToggleFavorite(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert a record
	metadata := createTestMetadata("toggle_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)

	// The initial state should be false
	box := store.FindRecords(rids)
	assert.False(t, box.Items[0].Favorite)

	// Toggle to true
	require.NoError(t, store.ToggleFavorite(rids[0]))

	box = store.FindRecords(rids)
	assert.True(t, box.Items[0].Favorite)

	// Toggle back to false
	require.NoError(t, store.ToggleFavorite(rids[0]))

	box = store.FindRecords(rids)
	assert.False(t, box.Items[0].Favorite)
}

func TestWithContext(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	ctx := context.WithValue(context.Background(), "test_key", "test_value")
	ctxStore := store.WithContext(ctx)

	assert.NotNil(t, ctxStore)

	// Verify context store works
	metadata := createTestMetadata("ctx_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, ctxStore.InsertRecord(metadata, importData))

	count := ctxStore.Count(resource.Filter{})
	assert.Equal(t, 1, count)
}

func TestWithTx_Success(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	err := store.WithTx(func(txStore record.TxStore) error {
		return txStore.InsertRecord(createTestMetadata("tx_test1"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 1,
		})
	})

	require.NoError(t, err)
	assert.Equal(t, 1, store.Count(resource.Filter{}))
}

func TestWithTx_Rollback(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	err := store.WithTx(func(txStore record.TxStore) error {
		if err := txStore.InsertRecord(createTestMetadata("tx_test2"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 2,
		}); err != nil {
			return err
		}
		return context.Canceled
	})

	assert.Error(t, err)
	assert.Equal(t, 0, store.Count(resource.Filter{}))
}

func TestWithTx_Nested(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	err := store.WithTx(func(txStore record.TxStore) error {
		if err := txStore.InsertRecord(createTestMetadata("nested1"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 10,
		}); err != nil {
			return err
		}

		return txStore.InsertRecord(createTestMetadata("nested2"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 11,
		})
	})

	require.NoError(t, err)
	assert.Equal(t, 2, store.Count(resource.Filter{}))
}

func TestUpsertTags(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert record with tags
	metadata := createTestMetadata("tag_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Insert another record with overlapping tags
	metadata2 := createTestMetadata("tag_test2")
	metadata2.Tags = []models.Tag{
		{Slug: "tag2", Name: "Tag 2"}, // Same as first record
		{Slug: "tag3", Name: "Tag 3"}, // New tag
	}
	importData2 := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 2,
	}
	require.NoError(t, store.InsertRecord(metadata2, importData2))

	// Verify records have correct tags
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)
	box := store.FindRecords(rids)

	assert.Len(t, box.Items[0].Tags, 2)
	assert.Len(t, box.Items[1].Tags, 2)
}

func TestUpsertCreator(t *testing.T) {
	store := setupStore(t)
	defer store.Close()

	// Insert record with creator
	metadata := createTestMetadata("creator_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, store.InsertRecord(metadata, importData))

	// Insert another record with the same creator but updated info
	metadata2 := createTestMetadata("creator_test2")
	metadata2.CreatorInfo.Nickname = "Updated Nickname"
	importData2 := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 2,
	}
	require.NoError(t, store.InsertRecord(metadata2, importData2))

	// Verify both records have updated creator info
	rids := store.FindPagedRIDs(resource.Filter{}, 0, 10)
	box := store.FindRecords(rids)

	for _, rec := range box.Items {
		assert.Equal(t, "Updated Nickname", rec.Nickname)
		assert.Equal(t, "creator_123", rec.Creator.PlatformID)
	}
}
