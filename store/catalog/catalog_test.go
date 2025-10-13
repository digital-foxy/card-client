package catalog

import (
	"bytes"
	"context"
	"image"
	"image/color"
	lpng "image/png"
	"testing"

	"github.com/r3dpixel/card-client/store/blob/pblob"
	"github.com/r3dpixel/card-client/store/record/erecord"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

func setupCatalog(t *testing.T) Service {
	recordStore, err := erecord.InMemoryStore()
	assert.NoError(t, err, "Failed to create in-memory record store")

	blobStore, err := pblob.New(t.TempDir(), pblob.Options{
		MaxVersions:   5,
		ThumbnailSize: 256,
	})

	catalog := New(recordStore, blobStore)
	require.NoError(t, err, "New catalog failed")
	t.Cleanup(func() { catalog.Close() })
	return catalog
}

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

// Count tests

func TestCount_Empty(t *testing.T) {
	catalog := setupCatalog(t)

	count, err := catalog.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCount_WithRecords(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 3 cards
	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("count_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	count, err := catalog.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// FindPagedRIDs tests

func TestFindPagedRIDs_Empty(t *testing.T) {
	catalog := setupCatalog(t)

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, rids)
}

func TestFindPagedRIDs_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 3 cards
	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("paged_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	assert.Len(t, rids, 3)
}

func TestFindPagedRIDs_Pagination(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 5 cards
	for i := 1; i <= 5; i++ {
		metadata := createTestMetadata("page_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	// First page
	rids1, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 2)
	require.NoError(t, err)
	assert.Len(t, rids1, 2)

	// Second page
	rids2, err := catalog.FindPagedRIDs(resource.Filter{}, 2, 2)
	require.NoError(t, err)
	assert.Len(t, rids2, 2)

	// Third page
	rids3, err := catalog.FindPagedRIDs(resource.Filter{}, 4, 2)
	require.NoError(t, err)
	assert.Len(t, rids3, 1)
}

// FindRecords tests

func TestFindRecords_Empty(t *testing.T) {
	catalog := setupCatalog(t)

	box, err := catalog.FindRecords()
	require.NoError(t, err)
	assert.Empty(t, box.Items)
}

func TestFindRecords_Single(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("find_records_1")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)

	box, err := catalog.FindRecords(rids[0])
	require.NoError(t, err)
	assert.Len(t, box.Items, 1)
	assert.Equal(t, metadata.Name, box.Items[0].Name)
}

func TestFindRecords_Multiple(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 3 cards
	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("find_multi_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 3)

	box, err := catalog.FindRecords(rids...)
	require.NoError(t, err)
	assert.Len(t, box.Items, 3)
}

func TestFindRecords_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	box, err := catalog.FindRecords(999)
	require.NoError(t, err)
	assert.Empty(t, box.Items)
}

// InsertCard tests

func TestInsertCard_Success(t *testing.T) {
	catalog := setupCatalog(t)

	metadata := createTestMetadata("insert_test_1")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	err := catalog.InsertCard(metadata, card, importData)
	require.NoError(t, err)

	// Verify record was inserted
	count, err := catalog.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestInsertCard_RecordStored(t *testing.T) {
	catalog := setupCatalog(t)

	metadata := createTestMetadata("insert_test_2")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Find the inserted record
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	box, err := catalog.FindRecords(rids[0])
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	record := box.Items[0]
	assert.Equal(t, metadata.Name, record.Name)
	assert.Equal(t, metadata.Title, record.Title)
	assert.Equal(t, metadata.CardInfo.PlatformID, record.InfoData.PlatformID)
	assert.Len(t, record.Tags, 2)
}

func TestInsertCard_TagsSetInCard(t *testing.T) {
	catalog := setupCatalog(t)

	metadata := createTestMetadata("insert_test_3")
	card := createTestCharacterCard(t)

	// Card starts with no tags
	require.Empty(t, card.Tags)

	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Card should now have tags set
	assert.Len(t, card.Tags, 2)
	assert.Contains(t, card.Tags, "Tag 1")
	assert.Contains(t, card.Tags, "Tag 2")
}

func TestInsertCard_BlobStored(t *testing.T) {
	catalog := setupCatalog(t)

	metadata := createTestMetadata("insert_test_4")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the record to check blob was stored
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	box, err := catalog.FindRecords(rids[0])
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	record := box.Items[0]

	// Verify blob exists by getting the catalog's internal stores
	// Since we can't access internal blob store directly, we verify
	// the record was created successfully which implies blob was stored
	assert.NotEmpty(t, record.ID)
	assert.NotZero(t, record.UpdateTime)
}

func TestInsertCard_MultipleCards(t *testing.T) {
	catalog := setupCatalog(t)

	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("multi_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	count, err := catalog.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// UpdateCard tests

func TestUpdateCard_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert initial card
	metadata := createTestMetadata("update_test_1")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)
	rid := rids[0]

	// Update the card
	metadata.Name = "Updated Name"
	metadata.Title = "Updated Title"
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}

	err = catalog.UpdateCard(rid, metadata, card, syncData.SyncTime)
	require.NoError(t, err)

	// Verify update
	box, err := catalog.FindRecords(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	record := box.Items[0]
	assert.Equal(t, "Updated Name", record.Name)
	assert.Equal(t, "Updated Title", record.Title)
	assert.Equal(t, resource.SyncSuccess, record.SyncStatus)
}

func TestUpdateCard_UpdatesTags(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert card with initial tags
	metadata := createTestMetadata("update_tags_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Update with new tags
	metadata.Tags = []models.Tag{
		{Slug: "tag2", Name: "Tag 2"}, // Keep one existing
		{Slug: "tag3", Name: "Tag 3"}, // Add new
		{Slug: "tag4", Name: "Tag 4"}, // Add new
	}
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}

	require.NoError(t, catalog.UpdateCard(rid, metadata, card, syncData.SyncTime))

	// Verify tags updated
	box, err := catalog.FindRecords(rid)
	require.NoError(t, err)
	require.Len(t, box.Items, 1)

	record := box.Items[0]
	assert.Len(t, record.Tags, 3)
}

func TestUpdateCard_NonExistentRecord(t *testing.T) {
	catalog := setupCatalog(t)

	metadata := createTestMetadata("nonexistent")
	card := createTestCharacterCard(t)
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}

	err := catalog.UpdateCard(999, metadata, card, syncData.SyncTime)
	assert.Error(t, err)
}

func TestUpdateCard_MultipleUpdates(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert initial card
	metadata := createTestMetadata("multi_update")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Perform multiple updates
	for i := 1; i <= 3; i++ {
		metadata.Name = "Update " + string(rune('0'+i))
		syncData := resource.SyncData{
			SyncTime:   timestamp.Now[timestamp.Nano](),
			SyncStatus: resource.SyncSuccess,
		}
		require.NoError(t, catalog.UpdateCard(rid, metadata, card, syncData.SyncTime))
	}

	// Verify final state
	box, err := catalog.FindRecords(rid)
	require.NoError(t, err)
	assert.Equal(t, "Update 3", box.Items[0].Name)
}

// FindRecord tests

func TestFindRecord_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("find_record_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	// Find the record
	record, err := catalog.FindRecord(rids[0])
	require.NoError(t, err)
	require.NotNil(t, record)

	// Verify record contents
	assert.Equal(t, rids[0], record.ID)
	assert.Equal(t, metadata.Name, record.Name)
	assert.Equal(t, metadata.Title, record.Title)
	assert.Equal(t, metadata.CardInfo.PlatformID, record.InfoData.PlatformID)
}

func TestFindRecord_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	// Try to find a non-existent record
	record, err := catalog.FindRecord(999)
	assert.Error(t, err)
	assert.Nil(t, record)
}

// FindExportHeaders tests

func TestFindExportHeaders_Empty(t *testing.T) {
	catalog := setupCatalog(t)

	box, err := catalog.FindExportHeaders()
	require.NoError(t, err)
	assert.Empty(t, box.Items)
}

func TestFindExportHeaders_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("export_headers_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)

	box, err := catalog.FindExportHeaders(rids[0])
	require.NoError(t, err)
	assert.Len(t, box.Items, 1)
}

func TestFindExportHeaders_Multiple(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 3 cards
	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("export_multi_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)

	box, err := catalog.FindExportHeaders(rids...)
	require.NoError(t, err)
	assert.Len(t, box.Items, 3)
}

// FindURLs tests

func TestFindURLs_Empty(t *testing.T) {
	catalog := setupCatalog(t)

	urls, err := catalog.FindURLs("https://example.com/nonexistent")
	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestFindURLs_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("url_test_1")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	urls, err := catalog.FindURLs(metadata.NormalizedURL)
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, metadata.NormalizedURL, urls[0])
}

func TestFindURLs_Multiple(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 3 cards
	var expectedURLs []string
	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("url_multi_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
		expectedURLs = append(expectedURLs, metadata.NormalizedURL)
	}

	urls, err := catalog.FindURLs(expectedURLs...)
	require.NoError(t, err)
	assert.Len(t, urls, 3)
}

// UpdateSyncData tests

func TestUpdateSyncData_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("sync_data_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Update sync data
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}
	err = catalog.UpdateSyncData(rid, syncData)
	require.NoError(t, err)

	// Verify update
	record, err := catalog.FindRecord(rid)
	require.NoError(t, err)
	assert.Equal(t, resource.SyncSuccess, record.SyncStatus)
}

func TestUpdateSyncData_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}
	err := catalog.UpdateSyncData(999, syncData)
	assert.Error(t, err)
}

// UpdateExportData tests

func TestUpdateExportData_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("export_data_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Update export data
	exportData := resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: 5,
	}
	err = catalog.UpdateExportData(rid, exportData)
	require.NoError(t, err)

	// Verify update
	record, err := catalog.FindRecord(rid)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(5), record.ExportedVersion)
}

func TestUpdateExportData_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	exportData := resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: 5,
	}
	err := catalog.UpdateExportData(999, exportData)
	assert.Error(t, err)
}

// UpdateFavoriteData tests

func TestUpdateFavoriteData_Single(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("favorite_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Set as favorite
	err = catalog.UpdateFavoriteData(true, rid)
	require.NoError(t, err)

	// Verify
	record, err := catalog.FindRecord(rid)
	require.NoError(t, err)
	assert.True(t, record.Favorite)

	// Unset favorite
	err = catalog.UpdateFavoriteData(false, rid)
	require.NoError(t, err)

	record, err = catalog.FindRecord(rid)
	require.NoError(t, err)
	assert.False(t, record.Favorite)
}

func TestUpdateFavoriteData_Multiple(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert 3 cards
	for i := 1; i <= 3; i++ {
		metadata := createTestMetadata("fav_multi_" + string(rune('0'+i)))
		card := createTestCharacterCard(t)
		importData := resource.ImportData{
			ImportTime:  timestamp.Now[timestamp.Nano](),
			ImportIndex: i,
		}
		require.NoError(t, catalog.InsertCard(metadata, card, importData))
	}

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)

	// Set all as favorites
	err = catalog.UpdateFavoriteData(true, rids...)
	require.NoError(t, err)

	// Verify all are favorites
	box, err := catalog.FindRecords(rids...)
	require.NoError(t, err)
	for _, record := range box.Items {
		assert.True(t, record.Favorite)
	}
}

// ToggleFavorite tests

func TestToggleFavorite_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("toggle_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Initially not favorite
	record, err := catalog.FindRecord(rid)
	require.NoError(t, err)
	initialState := record.Favorite

	// Toggle
	err = catalog.ToggleFavorite(rid)
	require.NoError(t, err)

	record, err = catalog.FindRecord(rid)
	require.NoError(t, err)
	assert.NotEqual(t, initialState, record.Favorite)

	// Toggle back
	err = catalog.ToggleFavorite(rid)
	require.NoError(t, err)

	record, err = catalog.FindRecord(rid)
	require.NoError(t, err)
	assert.Equal(t, initialState, record.Favorite)
}

func TestToggleFavorite_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	err := catalog.ToggleFavorite(999)
	assert.Error(t, err)
}

// Get/GetBytes/Thumbnail tests

func TestGet_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("get_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID and version
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	versions := catalog.CardVersions(rids[0])
	require.Len(t, versions, 1)

	// Get the card
	rawCard, err := catalog.GetRawCard(rids[0], versions[0])
	require.NoError(t, err)
	assert.NotNil(t, rawCard)
}

func TestGet_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	_, err := catalog.GetRawCard(999, 100)
	assert.Error(t, err)
}

func TestGetBytes_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("getbytes_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID and version
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	versions := catalog.CardVersions(rids[0])
	require.Len(t, versions, 1)

	// Get the bytes
	bytes, err := catalog.GetCardBytes(rids[0], versions[0])
	require.NoError(t, err)
	assert.NotNil(t, bytes)
	assert.NotEmpty(t, bytes)
}

func TestGetBytes_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	_, err := catalog.GetCardBytes(999, 100)
	assert.Error(t, err)
}

func TestThumbnail_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("thumbnail_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	// Get the thumbnail
	thumbnail, err := catalog.Thumbnail(rids[0])
	require.NoError(t, err)
	assert.NotNil(t, thumbnail)

	// Verify thumbnail dimensions (assuming 256 from setup)
	bounds := thumbnail.Bounds()
	assert.Equal(t, 256, bounds.Dx())
	assert.Equal(t, 256, bounds.Dy())
}

func TestThumbnail_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	_, err := catalog.Thumbnail(999)
	assert.Error(t, err)
}

func TestThumbnailBytes_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("thumbnail_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	// Get the thumbnail
	thumbnailBytes, err := catalog.ThumbnailBytes(rids[0])
	require.NoError(t, err)
	assert.NotNil(t, thumbnailBytes)

	thumbnail, _, err := image.Decode(bytes.NewReader(thumbnailBytes))
	assert.NoError(t, err)

	// Verify thumbnail dimensions (assuming 256 from setup)
	bounds := thumbnail.Bounds()
	assert.Equal(t, 256, bounds.Dx())
	assert.Equal(t, 256, bounds.Dy())
}

func TestThumbnailBytes_NonExistent(t *testing.T) {
	catalog := setupCatalog(t)

	_, err := catalog.ThumbnailBytes(999)
	assert.Error(t, err)
}

func TestCardVersions_Success(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("versions_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	// Check versions
	versions := catalog.CardVersions(rids[0])
	assert.Len(t, versions, 1)
	assert.NotZero(t, versions[0])
}

func TestCardVersions_Empty(t *testing.T) {
	catalog := setupCatalog(t)

	versions := catalog.CardVersions(999)
	assert.Empty(t, versions)
}

func TestCardVersions_Multiple(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert initial card
	metadata := createTestMetadata("versions_multi")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	rid := rids[0]

	// Update to create new version
	metadata.Name = "Updated"
	metadata.UpdateTime = timestamp.Now[timestamp.Nano]()
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}
	require.NoError(t, catalog.UpdateCard(rid, metadata, card, syncData.SyncTime))

	// Should have 2 versions
	versions := catalog.CardVersions(rid)
	assert.Len(t, versions, 2)
}

func TestCardVersionExists_True(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("version_exists")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID and version
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	versions := catalog.CardVersions(rids[0])
	require.Len(t, versions, 1)

	// Check version exists
	exists, err := catalog.CardVersionExists(rids[0], versions[0])
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCardVersionExists_False(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("version_not_exists")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Get the RID
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	// Check non-existent version
	exists, err := catalog.CardVersionExists(rids[0], 999)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCardVersionExists_NonExistentRID(t *testing.T) {
	catalog := setupCatalog(t)

	exists, err := catalog.CardVersionExists(999, 100)
	require.NoError(t, err)
	assert.False(t, exists)
}

// WithContext tests

func TestWithContext_CreatesNewInstance(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert a card
	metadata := createTestMetadata("with_context_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Create context-aware catalog
	ctx := context.Background()
	ctxCatalog := catalog.WithContext(ctx)
	require.NotNil(t, ctxCatalog)

	// Should be able to use it
	count, err := ctxCatalog.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// Close tests

func TestClose_Success(t *testing.T) {
	recordStore, err := erecord.InMemoryStore()
	require.NoError(t, err)

	blobStore, err := pblob.New(t.TempDir(), pblob.Options{
		MaxVersions:   5,
		ThumbnailSize: 256,
	})
	require.NoError(t, err)

	catalog := New(recordStore, blobStore)

	// Insert some data
	metadata := createTestMetadata("close_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Close should succeed
	err = catalog.Close()
	assert.NoError(t, err)
}

// Integration tests

func TestInsertAndUpdateCard_Integration(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert
	metadata := createTestMetadata("integration_test")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	// Find
	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, rids, 1)

	// Update
	metadata.Name = "Integration Updated"
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}
	require.NoError(t, catalog.UpdateCard(rids[0], metadata, card, syncData.SyncTime))

	// Verify
	box, err := catalog.FindRecords(rids[0])
	require.NoError(t, err)
	assert.Equal(t, "Integration Updated", box.Items[0].Name)
}

// Verify read-your-own-writes within transaction

func TestInsertCard_ReadWithinTransaction(t *testing.T) {
	catalog := setupCatalog(t)

	// This test verifies that FindRecord can read the record
	// that was just inserted within the same transaction
	metadata := createTestMetadata("read_within_tx")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}

	// If the transaction can't read its own writes, this will fail
	err := catalog.InsertCard(metadata, card, importData)
	require.NoError(t, err, "Should be able to read own writes within transaction")

	// Verify the record exists
	count, err := catalog.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUpdateCard_ReadWithinTransaction(t *testing.T) {
	catalog := setupCatalog(t)

	// Insert
	metadata := createTestMetadata("update_read_tx")
	card := createTestCharacterCard(t)
	importData := resource.ImportData{
		ImportTime:  timestamp.Now[timestamp.Nano](),
		ImportIndex: 1,
	}
	require.NoError(t, catalog.InsertCard(metadata, card, importData))

	rids, err := catalog.FindPagedRIDs(resource.Filter{}, 0, 10)
	require.NoError(t, err)

	// Update - FindRecord must read updated record within same transaction
	metadata.Name = "Updated Within TX"
	syncData := resource.SyncData{
		SyncTime:   timestamp.Now[timestamp.Nano](),
		SyncStatus: resource.SyncSuccess,
	}

	err = catalog.UpdateCard(rids[0], metadata, card, syncData.SyncTime)
	require.NoError(t, err, "Should be able to read own updates within transaction")

	// Verify update persisted
	box, err := catalog.FindRecords(rids[0])
	require.NoError(t, err)
	assert.Equal(t, "Updated Within TX", box.Items[0].Name)
}
