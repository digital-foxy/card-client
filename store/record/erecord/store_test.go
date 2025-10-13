package erecord

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
	store, err := InMemoryStore()
	require.NoError(t, err, "New failed")
	t.Cleanup(func() { store.Close() })
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

func TestCount(t *testing.T) {
	store := setupStore(t)

	// Test Count with an empty store
	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	_, err = store.InsertRecord(metadata, importData)
	assert.NoError(t, err)

	// Test Count with one record
	count, err = store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestFindPagedRIDs(t *testing.T) {
	store := setupStore(t)

	// Insert multiple records
	for i := 1; i <= 5; i++ {
		metadata := createTestMetadata(string(rune('0' + i)))
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		_, err := store.InsertRecord(metadata, importData)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Test pagination
	rids, err := store.FindPagedRIDs(resource.Filter{}, 0, 3)
	require.NoError(t, err)
	assert.Len(t, rids, 3)

	// Test offset
	rids, err = store.FindPagedRIDs(resource.Filter{}, 3, 3)
	require.NoError(t, err)
	assert.Len(t, rids, 2)

	// Test empty result
	rids, err = store.FindPagedRIDs(resource.Filter{}, 10, 3)
	require.NoError(t, err)
	assert.Empty(t, rids)
}

func TestFindRecord(t *testing.T) {
	store := setupStore(t)

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Find the record
	rec, err := store.FindRecord(rid)
	require.NoError(t, err)
	require.NotNil(t, rec)

	assert.Equal(t, metadata.Name, rec.Name)
	assert.Equal(t, metadata.Title, rec.Title)
	assert.Equal(t, metadata.CardInfo.PlatformID, rec.InfoData.PlatformID)
	assert.Len(t, rec.Tags, 2)
	assert.NotNil(t, rec.Creator)
}

func TestFindRecord_NotFound(t *testing.T) {
	store := setupStore(t)

	rec, err := store.FindRecord(0)
	assert.Error(t, err)
	assert.Nil(t, rec)
}

func TestFindRecords(t *testing.T) {
	store := setupStore(t)

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Find the record
	box, err := store.FindRecords(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	rec := box.Items[0]
	assert.Equal(t, metadata.Name, rec.Name)
	assert.Equal(t, metadata.Title, rec.Title)
	assert.Equal(t, metadata.CardInfo.PlatformID, rec.InfoData.PlatformID)
	assert.Len(t, rec.Tags, 2)
}

func TestFindRecords_Empty(t *testing.T) {
	store := setupStore(t)

	box, err := store.FindRecords()
	require.NoError(t, err)
	assert.Empty(t, box.Items)
}

func TestFindExportHeaders(t *testing.T) {
	store := setupStore(t)

	// Insert a record
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Update export data
	exportData := resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: timestamp.Now[timestamp.Nano](),
	}
	require.NoError(t, store.UpdateExportData(rid, exportData))

	// Find export headers
	box, err := store.FindExportHeaders(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	header := box.Items[0]
	assert.Equal(t, exportData.ExportTime, header.ExportTime)
}

func TestFindExportHeaders_Empty(t *testing.T) {
	store := setupStore(t)

	box, err := store.FindExportHeaders()
	require.NoError(t, err)
	assert.Empty(t, box.Items)
}

func TestFindURLs(t *testing.T) {
	store := setupStore(t)

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
		_, err := store.InsertRecord(metadata, importData)
		require.NoError(t, err)
	}

	// Find existing URLs
	searchURLs := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/nonexistent",
	}
	foundURLs, err := store.FindURLs(searchURLs...)
	require.NoError(t, err)

	assert.Len(t, foundURLs, 2)
	assert.Contains(t, foundURLs, "https://example.com/1")
	assert.Contains(t, foundURLs, "https://example.com/2")
}

func TestFindURLs_Empty(t *testing.T) {
	store := setupStore(t)

	foundURLs, err := store.FindURLs()
	require.NoError(t, err)
	assert.Empty(t, foundURLs)
}

func TestFindTagNames(t *testing.T) {
	store := setupStore(t)

	// Insert a record with tags
	metadata := createTestMetadata("test1")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Get record to extract TIDs
	box, err := store.FindRecords(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	rec := box.Items[0]
	require.Len(t, rec.Tags, 2)

	// Extract TIDs
	tids := make([]resource.TID, len(rec.Tags))
	for i, tag := range rec.Tags {
		tids[i] = tag.ID
	}

	// Find tag names
	names, err := store.FindTagNames(tids...)
	require.NoError(t, err)
	assert.Len(t, names, 2)
	assert.Contains(t, names, "Tag 1")
	assert.Contains(t, names, "Tag 2")
}

func TestFindTagNames_Empty(t *testing.T) {
	store := setupStore(t)

	names, err := store.FindTagNames()
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestFindTagNames_NonExistent(t *testing.T) {
	store := setupStore(t)

	names, err := store.FindTagNames("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, names)
}

// Insert/Update tests

func TestInsertRecord(t *testing.T) {
	store := setupStore(t)

	metadata := createTestMetadata("insert_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Verify insertion
	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify tags were inserted
	box, err := store.FindRecords(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)
	assert.Len(t, box.Items[0].Tags, 2)
}

func TestUpdateRecord(t *testing.T) {
	store := setupStore(t)

	// Insert a record with tags: tag1, tag2
	metadata := createTestMetadata("update_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

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

	require.NoError(t, store.UpdateRecord(rid, metadata, syncData.SyncTime))

	// Verify update
	box, err := store.FindRecords(rid)
	require.NoError(t, err)
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

	// Insert a record
	metadata := createTestMetadata("sync_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Update sync data
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncFailed,
	}
	require.NoError(t, store.UpdateSyncData(rid, syncData))

	// Verify update
	box, err := store.FindRecords(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	assert.Equal(t, resource.SyncFailed, box.Items[0].SyncStatus)
}

func TestUpdateExportData(t *testing.T) {
	store := setupStore(t)

	// Insert a record
	metadata := createTestMetadata("export_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Update export data
	exportData := resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: timestamp.Now[timestamp.Nano](),
	}
	require.NoError(t, store.UpdateExportData(rid, exportData))

	// Verify update
	box, err := store.FindExportHeaders(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	assert.Equal(t, exportData.ExportTime, box.Items[0].ExportTime)
}

func TestUpdateFavoriteData(t *testing.T) {
	store := setupStore(t)

	// Insert multiple records
	var rids []resource.RID
	for i := 0; i < 3; i++ {
		metadata := createTestMetadata(string(rune('1' + i)))
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i + 1,
		}
		rid, err := store.InsertRecord(metadata, importData)
		require.NoError(t, err)
		rids = append(rids, rid)
	}

	// Update favorite data
	require.NoError(t, store.UpdateFavoriteData(true, rids...))

	// Verify update
	box, err := store.FindRecords(rids...)
	require.NoError(t, err)
	for _, rec := range box.Items {
		assert.True(t, rec.Favorite)
	}

	// Unfavorite
	require.NoError(t, store.UpdateFavoriteData(false, rids[:2]...))

	box, err = store.FindRecords(rids...)
	require.NoError(t, err)
	assert.False(t, box.Items[0].Favorite)
	assert.False(t, box.Items[1].Favorite)
	assert.True(t, box.Items[2].Favorite)
}

func TestToggleFavorite(t *testing.T) {
	store := setupStore(t)

	// Insert a record
	metadata := createTestMetadata("toggle_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// The initial state should be false
	box, err := store.FindRecords(rid)
	require.NoError(t, err)
	assert.False(t, box.Items[0].Favorite)

	// Toggle to true
	require.NoError(t, store.ToggleFavorite(rid))

	box, err = store.FindRecords(rid)
	require.NoError(t, err)
	assert.True(t, box.Items[0].Favorite)

	// Toggle back to false
	require.NoError(t, store.ToggleFavorite(rid))

	box, err = store.FindRecords(rid)
	require.NoError(t, err)
	assert.False(t, box.Items[0].Favorite)
}

// Context and transaction tests

func TestWithContext(t *testing.T) {
	store := setupStore(t)

	ctx := context.WithValue(context.Background(), "test_key", "test_value")
	ctxStore := store.WithContext(ctx)

	assert.NotNil(t, ctxStore)

	// Verify context store works
	metadata := createTestMetadata("ctx_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	_, err := ctxStore.InsertRecord(metadata, importData)
	require.NoError(t, err)

	count, err := ctxStore.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWithTx_Commit(t *testing.T) {
	store := setupStore(t)

	err := store.WithTx(func(txStore record.TxStore) error {
		_, err := txStore.InsertRecord(createTestMetadata("tx_test1"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 1,
		})
		return err
	})

	require.NoError(t, err)

	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWithTx_Rollback(t *testing.T) {
	store := setupStore(t)

	err := store.WithTx(func(txStore record.TxStore) error {
		_, err := txStore.InsertRecord(createTestMetadata("tx_test2"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 2,
		})
		if err != nil {
			return err
		}
		return context.Canceled
	})

	assert.Error(t, err)

	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestWithTx_Nested(t *testing.T) {
	store := setupStore(t)

	err := store.WithTx(func(txStore record.TxStore) error {
		_, err := txStore.InsertRecord(createTestMetadata("nested1"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 10,
		})
		if err != nil {
			return err
		}

		_, err = txStore.InsertRecord(createTestMetadata("nested2"), resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: 11,
		})
		return err
	})

	require.NoError(t, err)

	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// Tag and creator tests

func TestUpsertTags(t *testing.T) {
	store := setupStore(t)

	// Insert record with tags
	metadata := createTestMetadata("tag_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid1, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

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
	rid2, err := store.InsertRecord(metadata2, importData2)
	require.NoError(t, err)

	// Verify records have correct tags
	box, err := store.FindRecords(rid1, rid2)
	require.NoError(t, err)

	assert.Len(t, box.Items[0].Tags, 2)
	assert.Len(t, box.Items[1].Tags, 2)
}

func TestUpsertCreator(t *testing.T) {
	store := setupStore(t)

	// Insert record with creator
	metadata := createTestMetadata("creator_test")
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	rid1, err := store.InsertRecord(metadata, importData)
	require.NoError(t, err)

	// Insert another record with the same creator but updated info
	metadata2 := createTestMetadata("creator_test2")
	metadata2.CreatorInfo.Nickname = "Updated Nickname"
	importData2 := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 2,
	}
	rid2, err := store.InsertRecord(metadata2, importData2)
	require.NoError(t, err)

	// Verify both records have updated creator info
	box, err := store.FindRecords(rid1, rid2)
	require.NoError(t, err)

	for _, rec := range box.Items {
		assert.Equal(t, "Updated Nickname", rec.Nickname)
		assert.Equal(t, "creator_123", rec.Creator.PlatformID)
	}
}
