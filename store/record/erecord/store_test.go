package erecord

import (
	"context"
	"testing"
	"time"

	"github.com/digital-foxy/card-client/store/record"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/recordentity"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/timestamp"
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
	now := timestamp.NowNano()
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

func createTestCharacterCard(t *testing.T) *png.CharacterCard {
	t.Helper()
	return createCard("Test description", "Test personality", "Test scenario", "Hello", "Examples", "Notes", "Prompt", "History", []string{"Hi"})
}

func createCard(description, personality, scenario, firstMessage, messageExamples, creatorNotes, systemPrompt, postHistory string, greetings []string) *png.CharacterCard {
	card := &png.CharacterCard{
		Sheet: &character.Sheet{},
	}
	card.Description.SetIf(description)
	card.Personality.SetIf(personality)
	card.Scenario.SetIf(scenario)
	card.FirstMessage.SetIf(firstMessage)
	card.MessageExamples.SetIf(messageExamples)
	card.CreatorNotes.SetIf(creatorNotes)
	card.SystemPrompt.SetIf(systemPrompt)
	card.PostHistoryInstructions.SetIf(postHistory)
	card.AlternateGreetings = greetings
	return card
}

func TestStoreClose(t *testing.T) {
	// Create a temporary directory for the database file
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"

	// Create a file-based store (not in-memory)
	store, err := New(dbPath, Options{
		CacheConnections:   true,
		MaxConnections:     2,
		MaxIdleConnections: 1,
		MaxLifetime:        0,
	})
	require.NoError(t, err, "Failed to create store")

	// Insert a record to verify the store is working
	metadata := createTestMetadata("close_test")
	characterCard := createTestCharacterCard(t)
	importTime := timestamp.NowNano()
	rid, err := store.SaveRecord(metadata, characterCard, importTime, 1)
	require.NoError(t, err, "Failed to insert record")

	// Verify the record was inserted
	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Close the store
	err = store.Close()
	assert.NoError(t, err, "Close should not return an error")

	// Verify operations fail after close
	_, err = store.FindRecord(rid)
	assert.Error(t, err, "Operations should fail after close")
}

func TestCount(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Test Count with an empty store
	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Insert a record
	metadata := createTestMetadata("test1")
	_, err = store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	assert.NoError(t, err)

	// Test Count with one record
	count, err = store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestFindPagedRIDs(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Insert multiple records
	for i := 1; i <= 5; i++ {
		metadata := createTestMetadata(string(rune('0' + i)))
		_, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), i)
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
	characterCard := createTestCharacterCard(t)

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, store record.Store) resource.RID
		expectError  bool
		validateFunc func(t *testing.T, rec *resource.Record, metadata *models.Metadata)
	}{
		{
			name: "success - find existing record",
			setupFunc: func(t *testing.T, store record.Store) resource.RID {
				metadata := createTestMetadata("test1")
				rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
				require.NoError(t, err)
				return rid
			},
			expectError: false,
			validateFunc: func(t *testing.T, rec *resource.Record, metadata *models.Metadata) {
				require.NotNil(t, rec)
				assert.Equal(t, metadata.Name, rec.Name)
				assert.Equal(t, metadata.Title, rec.Title)
				assert.Equal(t, metadata.CardInfo.PlatformID, rec.InfoData.PlatformID)
				assert.Len(t, rec.Tags, 2)
				assert.NotNil(t, rec.Creator)
			},
		},
		{
			name: "not found - nonexistent RID",
			setupFunc: func(t *testing.T, store record.Store) resource.RID {
				return 0 // No setup needed
			},
			expectError: true,
			validateFunc: func(t *testing.T, rec *resource.Record, metadata *models.Metadata) {
				assert.Nil(t, rec)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupStore(t)
			metadata := createTestMetadata("test1")
			rid := tt.setupFunc(t, store)

			rec, err := store.FindRecord(rid)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.validateFunc(t, rec, metadata)
		})
	}
}

func TestFindRecords(t *testing.T) {
	characterCard := createTestCharacterCard(t)

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, store record.Store) []resource.RID
		expectError  bool
		validateFunc func(t *testing.T, box resource.Box[resource.Record], metadata *models.Metadata)
	}{
		{
			name: "success - find existing record",
			setupFunc: func(t *testing.T, store record.Store) []resource.RID {
				metadata := createTestMetadata("test1")
				rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
				require.NoError(t, err)
				return []resource.RID{rid}
			},
			expectError: false,
			validateFunc: func(t *testing.T, box resource.Box[resource.Record], metadata *models.Metadata) {
				require.Len(t, box.Items, 1)
				rec := box.Items[0]
				assert.Equal(t, metadata.Name, rec.Name)
				assert.Equal(t, metadata.Title, rec.Title)
				assert.Equal(t, metadata.CardInfo.PlatformID, rec.InfoData.PlatformID)
				assert.Len(t, rec.Tags, 2)
			},
		},
		{
			name: "empty - no records",
			setupFunc: func(t *testing.T, store record.Store) []resource.RID {
				return []resource.RID{} // No setup needed
			},
			expectError: false,
			validateFunc: func(t *testing.T, box resource.Box[resource.Record], metadata *models.Metadata) {
				assert.Empty(t, box.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupStore(t)
			metadata := createTestMetadata("test1")
			rids := tt.setupFunc(t, store)

			box, err := store.FindRecords(rids...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.validateFunc(t, box, metadata)
		})
	}
}

func TestFindURLs(t *testing.T) {
	characterCard := createTestCharacterCard(t)

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, store record.Store)
		searchURLs   []string
		expectError  bool
		validateFunc func(t *testing.T, foundURLs []string)
	}{
		{
			name: "success - find existing URLs",
			setupFunc: func(t *testing.T, store record.Store) {
				urls := []string{
					"https://example.com/1",
					"https://example.com/2",
					"https://example.com/3",
				}
				for i, url := range urls {
					metadata := createTestMetadata(string(rune('1' + i)))
					metadata.NormalizedURL = url
					_, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), i+1)
					require.NoError(t, err)
				}
			},
			searchURLs: []string{
				"https://example.com/1",
				"https://example.com/2",
				"https://example.com/nonexistent",
			},
			expectError: false,
			validateFunc: func(t *testing.T, foundURLs []string) {
				assert.Len(t, foundURLs, 2)
				assert.Contains(t, foundURLs, "https://example.com/1")
				assert.Contains(t, foundURLs, "https://example.com/2")
			},
		},
		{
			name: "empty - no URLs to search",
			setupFunc: func(t *testing.T, store record.Store) {
				// No setup needed
			},
			searchURLs:  []string{},
			expectError: false,
			validateFunc: func(t *testing.T, foundURLs []string) {
				assert.Empty(t, foundURLs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupStore(t)
			tt.setupFunc(t, store)

			foundURLs, err := store.FindURLs(tt.searchURLs...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.validateFunc(t, foundURLs)
		})
	}
}

func TestFindTagNames(t *testing.T) {
	characterCard := createTestCharacterCard(t)

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, store record.Store) []resource.TID
		expectError  bool
		validateFunc func(t *testing.T, names []string)
	}{
		{
			name: "success - find existing tag names",
			setupFunc: func(t *testing.T, store record.Store) []resource.TID {
				metadata := createTestMetadata("test1")
				rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
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
				return tids
			},
			expectError: false,
			validateFunc: func(t *testing.T, names []string) {
				assert.Len(t, names, 2)
				assert.Contains(t, names, "Tag 1")
				assert.Contains(t, names, "Tag 2")
			},
		},
		{
			name: "empty - no TIDs to search",
			setupFunc: func(t *testing.T, store record.Store) []resource.TID {
				return []resource.TID{} // No setup needed
			},
			expectError: false,
			validateFunc: func(t *testing.T, names []string) {
				assert.Empty(t, names)
			},
		},
		{
			name: "empty - nonexistent TIDs",
			setupFunc: func(t *testing.T, store record.Store) []resource.TID {
				return []resource.TID{"nonexistent"}
			},
			expectError: false,
			validateFunc: func(t *testing.T, names []string) {
				assert.Empty(t, names)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupStore(t)
			tids := tt.setupFunc(t, store)

			names, err := store.FindTagNames(tids...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.validateFunc(t, names)
		})
	}
}

// Insert/Update tests

func TestSaveRecord(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	metadata := createTestMetadata("insert_test")
	rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
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
	characterCard := createTestCharacterCard(t)

	// Insert a record with tags: tag1, tag2
	metadata := createTestMetadata("update_test")
	rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	// Update the record with tags: tag2 (existing), tag3 (new), tag4 (new)
	metadata.Name = "Updated Name"
	metadata.Title = "Updated Title"
	metadata.Tags = []models.Tag{
		{Slug: "tag2", Name: "Tag 2"}, // Existing tag
		{Slug: "tag3", Name: "Tag 3"}, // New tag
		{Slug: "tag4", Name: "Tag 4"}, // New tag
	}

	_, err = store.SaveRecord(metadata, characterCard, timestamp.NowNano())
	require.NoError(t, err)

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
	characterCard := createTestCharacterCard(t)

	// Insert a record
	metadata := createTestMetadata("sync_test")
	rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	// Update sync data
	syncData := resource.SyncData{
		SyncTime:   timestamp.NowNano(),
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
	characterCard := createTestCharacterCard(t)

	// Insert a record
	metadata := createTestMetadata("export_test")
	rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	// Update export data
	exportData := resource.ExportData{
		ExportTime:      timestamp.NowNano(),
		ExportedVersion: timestamp.NowNano(),
	}
	require.NoError(t, store.UpdateExportData(rid, exportData))

	// Verify update by reading the record
	rec, err := store.FindRecord(rid)
	require.NoError(t, err)
	require.NotNil(t, rec)
}

func TestUpdateFavoriteData(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Insert multiple records
	var rids []resource.RID
	for i := 0; i < 3; i++ {
		metadata := createTestMetadata(string(rune('1' + i)))
		rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), i+1)
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
	characterCard := createTestCharacterCard(t)

	// Insert a record
	metadata := createTestMetadata("toggle_test")
	rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
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
	characterCard := createTestCharacterCard(t)

	ctx := context.WithValue(context.Background(), "test_key", "test_value")
	ctxStore := store.WithContext(ctx)

	assert.NotNil(t, ctxStore)

	// Verify context store works
	metadata := createTestMetadata("ctx_test")
	_, err := ctxStore.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	count, err := ctxStore.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWithTx_Commit(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	err := store.WithTx(func(txStore record.TxStore) error {
		_, err := txStore.SaveRecord(createTestMetadata("tx_test1"), characterCard, timestamp.NowNano(), 1)
		return err
	})

	require.NoError(t, err)

	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestWithTx_Rollback(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	err := store.WithTx(func(txStore record.TxStore) error {
		_, err := txStore.SaveRecord(createTestMetadata("tx_test2"), characterCard, timestamp.NowNano(), 2)
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
	characterCard := createTestCharacterCard(t)

	err := store.WithTx(func(txStore record.TxStore) error {
		_, err := txStore.SaveRecord(createTestMetadata("nested1"), characterCard, timestamp.NowNano(), 10)
		if err != nil {
			return err
		}

		_, err = txStore.SaveRecord(createTestMetadata("nested2"), characterCard, timestamp.NowNano(), 11)
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
	characterCard := createTestCharacterCard(t)

	// Insert record with tags
	metadata := createTestMetadata("tag_test")
	rid1, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	// Insert another record with overlapping tags
	metadata2 := createTestMetadata("tag_test2")
	metadata2.Tags = []models.Tag{
		{Slug: "tag2", Name: "Tag 2"}, // Same as first record
		{Slug: "tag3", Name: "Tag 3"}, // New tag
	}
	rid2, err := store.SaveRecord(metadata2, characterCard, timestamp.NowNano(), 2)
	require.NoError(t, err)

	// Verify records have correct tags
	box, err := store.FindRecords(rid1, rid2)
	require.NoError(t, err)

	assert.Len(t, box.Items[0].Tags, 2)
	assert.Len(t, box.Items[1].Tags, 2)
}

func TestUpsertCreator(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Insert record with creator
	metadata := createTestMetadata("creator_test")
	rid1, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	// Insert another record with the same creator but updated info
	metadata2 := createTestMetadata("creator_test2")
	metadata2.CreatorInfo.Nickname = "Updated Nickname"
	rid2, err := store.SaveRecord(metadata2, characterCard, timestamp.NowNano(), 2)
	require.NoError(t, err)

	// Verify both records have updated creator info
	box, err := store.FindRecords(rid1, rid2)
	require.NoError(t, err)

	for _, rec := range box.Items {
		assert.Equal(t, "Updated Nickname", rec.Nickname)
		assert.Equal(t, "creator_123", rec.Creator.PlatformID)
	}
}

func TestFindCreator(t *testing.T) {
	characterCard := createTestCharacterCard(t)

	tests := []struct {
		name         string
		setupFunc    func(t *testing.T, store record.Store) (resource.CID, *models.Metadata)
		expectError  bool
		validateFunc func(t *testing.T, creator resource.Creator, metadata *models.Metadata)
	}{
		{
			name: "success - find existing creator by CID",
			setupFunc: func(t *testing.T, store record.Store) (resource.CID, *models.Metadata) {
				metadata := createTestMetadata("creator_find_test")
				rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
				require.NoError(t, err)

				// Get the record to extract creator ID
				box, err := store.FindRecords(rid)
				require.NoError(t, err)
				require.Len(t, box.Items, 1)
				require.NotNil(t, box.Items[0].Creator)

				return box.Items[0].Creator.ID, metadata
			},
			expectError: false,
			validateFunc: func(t *testing.T, creator resource.Creator, metadata *models.Metadata) {
				assert.NotEmpty(t, creator.ID)
				assert.Equal(t, metadata.CreatorInfo.Nickname, creator.Nickname)
				assert.Equal(t, metadata.CreatorInfo.Username, creator.Username)
				assert.Equal(t, metadata.CreatorInfo.PlatformID, creator.PlatformID)
				assert.Equal(t, metadata.Source, creator.Source)
			},
		},
		{
			name: "not found - nonexistent CID",
			setupFunc: func(t *testing.T, store record.Store) (resource.CID, *models.Metadata) {
				return "nonexistent_creator", nil // No setup needed
			},
			expectError: true,
			validateFunc: func(t *testing.T, creator resource.Creator, metadata *models.Metadata) {
				assert.Empty(t, creator.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupStore(t)
			creatorID, metadata := tt.setupFunc(t, store)

			// Find the creator by ID
			creator, err := store.FindCreatorByCID(creatorID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.validateFunc(t, creator, metadata)
		})
	}
}

func TestFindCreatorByNickname(t *testing.T) {
	characterCard := createTestCharacterCard(t)

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, store record.Store) *models.Metadata
		searchSource   source.ID
		searchNickname string
		expectError    bool
		validateFunc   func(t *testing.T, creator resource.Creator, metadata *models.Metadata)
	}{
		{
			name: "success - find existing creator",
			setupFunc: func(t *testing.T, store record.Store) *models.Metadata {
				metadata := createTestMetadata("creator_nickname_test")
				rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
				require.NoError(t, err)

				// Verify creator was inserted
				box, err := store.FindRecords(rid)
				require.NoError(t, err)
				require.Len(t, box.Items, 1)
				require.NotNil(t, box.Items[0].Creator)

				return metadata
			},
			searchSource:   source.ChubAI,
			searchNickname: "Creator Nickname",
			expectError:    false,
			validateFunc: func(t *testing.T, creator resource.Creator, metadata *models.Metadata) {
				assert.Equal(t, metadata.CreatorInfo.Nickname, creator.Nickname)
				assert.Equal(t, metadata.CreatorInfo.Username, creator.Username)
				assert.Equal(t, metadata.CreatorInfo.PlatformID, creator.PlatformID)
				assert.Equal(t, metadata.Source, creator.Source)
			},
		},
		{
			name: "not found - nonexistent nickname",
			setupFunc: func(t *testing.T, store record.Store) *models.Metadata {
				return nil // No setup needed
			},
			searchSource:   source.ChubAI,
			searchNickname: "nonexistent_nickname",
			expectError:    true,
			validateFunc: func(t *testing.T, creator resource.Creator, metadata *models.Metadata) {
				assert.Empty(t, creator.ID)
			},
		},
		{
			name: "not found - wrong source",
			setupFunc: func(t *testing.T, store record.Store) *models.Metadata {
				metadata := createTestMetadata("creator_source_test")
				_, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
				require.NoError(t, err)
				return metadata
			},
			searchSource:   source.CharacterTavern,
			searchNickname: "Creator Nickname",
			expectError:    true,
			validateFunc: func(t *testing.T, creator resource.Creator, metadata *models.Metadata) {
				assert.Empty(t, creator.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupStore(t)
			metadata := tt.setupFunc(t, store)

			// Find the creator by nickname
			creator, err := store.FindCreatorByNickname(tt.searchSource, tt.searchNickname)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.validateFunc(t, creator, metadata)
		})
	}
}

func TestRegexFilter(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Insert test records with various titles to test regex filtering
	testRecords := []struct {
		platformID string
		title      string
	}{
		{"1", "hello world"},
		{"2", "Hello World"},
		{"3", "HELLO"},
		{"4", "hello"},
		{"5", "test hello test"},
		{"6", "world"},
		{"7", "Hello.World"},
		{"8", "[Hello]"},
	}

	for _, tc := range testRecords {
		metadata := createTestMetadata(tc.platformID)
		metadata.Title = tc.title
		_, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
	}

	tests := []struct {
		name          string
		value         string
		caseSensitive bool
		regex         bool
		wholeWord     bool
		expectedCount int
		description   string
	}{
		{
			name:          "Default case-insensitive contains",
			value:         "hello",
			caseSensitive: false,
			regex:         false,
			wholeWord:     false,
			expectedCount: 7, // All records with "hello" in any case
			description:   "Should match: hello world, Hello World, HELLO, hello, test hello test, Hello.World, [Hello]",
		},
		{
			name:          "Case-sensitive contains",
			value:         "hello",
			caseSensitive: true,
			regex:         false,
			wholeWord:     false,
			expectedCount: 3, // Only lowercase "hello"
			description:   "Should match: hello world, hello, test hello test",
		},
		{
			name:          "Whole word case-insensitive",
			value:         "hello",
			caseSensitive: false,
			regex:         false,
			wholeWord:     true,
			expectedCount: 7, // Word boundaries treat punctuation as boundaries
			description:   "Should match: hello world, Hello World, HELLO, hello, test hello test, Hello.World, [Hello]",
		},
		{
			name:          "Whole word case-sensitive",
			value:         "hello",
			caseSensitive: true,
			regex:         false,
			wholeWord:     true,
			expectedCount: 3, // Word boundaries, lowercase only
			description:   "Should match: hello world, hello, test hello test",
		},
		{
			name:          "Regex pattern case-insensitive",
			value:         "^[Hh]ello",
			caseSensitive: false,
			regex:         true,
			wholeWord:     false,
			expectedCount: 5, // Starts with hello (any case), excluding [Hello]
			description:   "Should match: hello world, Hello World, HELLO, hello, Hello.World",
		},
		{
			name:          "Regex pattern case-sensitive",
			value:         "^hello",
			caseSensitive: true,
			regex:         true,
			wholeWord:     false,
			expectedCount: 2, // Starts with lowercase "hello"
			description:   "Should match: hello world, hello",
		},
		{
			name:          "Regex with word boundaries",
			value:         "world$",
			caseSensitive: false,
			regex:         true,
			wholeWord:     false,
			expectedCount: 4, // Ends with "world" (any case)
			description:   "Should match: hello world, Hello World, world, Hello.World",
		},
		{
			name:          "Special chars escaped (not regex mode)",
			value:         "[Hello]",
			caseSensitive: false,
			regex:         false,
			wholeWord:     false,
			expectedCount: 1, // Literal brackets
			description:   "Should match: [Hello]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := resource.Filter{
				TextFilter: []resource.TextFilter{
					{
						Field: resource.FieldRecordTitle,
						Value: tt.value,
						MatchMode: resource.TextMatchMode{
							CaseSensitive: tt.caseSensitive,
							Regex:         tt.regex,
							WholeWord:     tt.wholeWord,
						},
					},
				},
			}

			count, err := store.Count(filter)
			require.NoError(t, err, "Count should not error for %s", tt.name)
			assert.Equal(t, tt.expectedCount, count, "%s: %s", tt.name, tt.description)
		})
	}
}

func TestCleanupCreators(t *testing.T) {
	store := setupStore(t).(*Store)
	characterCard := createTestCharacterCard(t)

	// Insert records with different creators
	metadata1 := createTestMetadata("cleanup1")
	metadata1.CreatorInfo.PlatformID = "creator_1"
	metadata1.CreatorInfo.Nickname = "Creator One"

	metadata2 := createTestMetadata("cleanup2")
	metadata2.CreatorInfo.PlatformID = "creator_2"
	metadata2.CreatorInfo.Nickname = "Creator Two"

	metadata3 := createTestMetadata("cleanup3")
	metadata3.CreatorInfo.PlatformID = "creator_1" // Same creator as record 1
	metadata3.CreatorInfo.Nickname = "Creator One"

	importTime := timestamp.NowNano()

	rid1, err := store.SaveRecord(metadata1, characterCard, importTime, 1)
	require.NoError(t, err)

	rid2, err := store.SaveRecord(metadata2, characterCard, importTime, 1)
	require.NoError(t, err)

	_, err = store.SaveRecord(metadata3, characterCard, importTime, 1)
	require.NoError(t, err)

	// Verify we have 2 creators (creator_1 and creator_2)
	cid1 := store.CID(source.ChubAI, "creator_1")
	cid2 := store.CID(source.ChubAI, "creator_2")

	_, err = store.FindCreatorByCID(cid1)
	require.NoError(t, err)
	_, err = store.FindCreatorByCID(cid2)
	require.NoError(t, err)

	// No orphaned creators yet
	deleted, err := store.CleanupCreators()
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)

	// Delete record 2 (leaving creator_2 orphaned)
	_, err = store.client.RecordEntity.Delete().Where(recordentity.ID(rid2)).Exec(store.ctx)
	require.NoError(t, err)

	// Now cleanup should delete creator_2
	deleted, err = store.CleanupCreators()
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// creator_2 should no longer exist
	_, err = store.FindCreatorByCID(cid2)
	assert.Error(t, err)

	// creator_1 should still exist (still referenced by records 1 and 3)
	_, err = store.FindCreatorByCID(cid1)
	require.NoError(t, err)

	// Delete record 1 (creator_1 still referenced by record 3)
	_, err = store.client.RecordEntity.Delete().Where(recordentity.ID(rid1)).Exec(store.ctx)
	require.NoError(t, err)

	// Cleanup should not delete creator_1 yet
	deleted, err = store.CleanupCreators()
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)

	// creator_1 should still exist
	_, err = store.FindCreatorByCID(cid1)
	require.NoError(t, err)
}

func TestDeleteWithFTSCleanup(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Insert a record (SaveRecord now handles FTS internally)
	metadata := createTestMetadata("delete_fts_test")
	rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), 1)
	require.NoError(t, err)

	// Verify record exists
	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Delete the record (FTS cleanup happens via trigger)
	deleted, err := store.Delete(rid)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify record is deleted
	count, err = store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDeleteMultipleRecords(t *testing.T) {
	store := setupStore(t)
	characterCard := createTestCharacterCard(t)

	// Insert multiple records (SaveRecord now handles FTS internally)
	var rids []resource.RID
	for i := 0; i < 3; i++ {
		metadata := createTestMetadata(string(rune('a' + i)))
		rid, err := store.SaveRecord(metadata, characterCard, timestamp.NowNano(), i+1)
		require.NoError(t, err)

		rids = append(rids, rid)
	}

	// Verify all records exist
	count, err := store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Delete first two records
	deleted, err := store.Delete(rids[0], rids[1])
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)

	// Verify only one record remains
	count, err = store.Count(resource.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestContentFilter(t *testing.T) {
	store := setupStore(t)

	testCases := []struct {
		platformID string
		card       *png.CharacterCard
	}{
		{
			platformID: "1",
			card:       createCard("A powerful wizard with magic abilities", "Wise and mysterious", "In a magical tower", "Greetings traveler", "Example dialogue", "This is a fantasy character", "You are a wizard", "Stay in character", []string{"Hello", "Welcome"}),
		},
		{
			platformID: "2",
			card:       createCard("A brave knight with sword skills", "Honorable and brave", "In a medieval castle", "Hail, stranger", "Combat examples", "This is a medieval character", "You are a knight", "Be honorable", []string{"Good day", "Greetings"}),
		},
		{
			platformID: "3",
			card:       createCard("A cunning rogue with stealth magic", "Sly and clever", "In the shadows", "Psst, over here", "Sneaky dialogue", "This is a rogue character", "You are a rogue", "Stay hidden", []string{"Hey", "Shh"}),
		},
	}

	// Insert records (SaveRecord now handles FTS internally)
	for _, tc := range testCases {
		metadata := createTestMetadata(tc.platformID)
		_, err := store.SaveRecord(metadata, tc.card, timestamp.NowNano(), 1)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		filter        resource.ContentFilter
		expectedCount int
		description   string
	}{
		{
			name: "Single value, single field (description) - match",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentDescription},
				Values:    []string{"wizard"},
				FieldMode: resource.ANY,
				ValueMode: resource.ANY,
			},
			expectedCount: 1,
			description:   "Should find record 1 (wizard in description)",
		},
		{
			name: "Single value, multiple fields (description, personality) - ANY field",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentDescription, resource.FieldContentPersonality},
				Values:    []string{"magic"},
				FieldMode: resource.ANY,
				ValueMode: resource.ANY,
			},
			expectedCount: 2,
			description:   "Should find record 1 (magic in description) and record 3 (magic in description)",
		},
		{
			name: "Multiple values (wizard, knight) - ANY value, ANY field",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentDescription},
				Values:    []string{"wizard", "knight"},
				FieldMode: resource.ANY,
				ValueMode: resource.ANY,
			},
			expectedCount: 2,
			description:   "Should find record 1 (wizard) and record 2 (knight)",
		},
		{
			name: "Multiple values (magic, stealth) - ALL values, single field",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentDescription},
				Values:    []string{"magic", "stealth"},
				FieldMode: resource.ANY,
				ValueMode: resource.ALL,
			},
			expectedCount: 1,
			description:   "Should find record 3 (has both 'stealth' and 'magic' in description)",
		},
		{
			name: "Single value across ALL fields",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentDescription, resource.FieldContentPersonality, resource.FieldContentScenario},
				Values:    []string{"character"},
				FieldMode: resource.ALL,
				ValueMode: resource.ANY,
			},
			expectedCount: 0,
			description:   "Should find no records (no record has 'character' in all three fields)",
		},
		{
			name: "Single value across ALL fields - positive match",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentCreatorNotes, resource.FieldContentScenario},
				Values:    []string{"character", "In"},
				FieldMode: resource.ALL,
				ValueMode: resource.ANY,
			},
			expectedCount: 3,
			description:   "Should find all 3 records (all have 'character' in creator_notes and scenario contains 'character' substring)",
		},
		{
			name: "Search in system_prompt field",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentSystemPrompt},
				Values:    []string{"wizard"},
				FieldMode: resource.ANY,
				ValueMode: resource.ANY,
			},
			expectedCount: 1,
			description:   "Should find record 1 (has 'wizard' in system_prompt)",
		},
		{
			name: "Multiple values - ALL values in ANY field",
			filter: resource.ContentFilter{
				Fields:    []string{resource.FieldContentDescription, resource.FieldContentCreatorNotes},
				Values:    []string{"medieval", "character"},
				FieldMode: resource.ANY,
				ValueMode: resource.ALL,
			},
			expectedCount: 1,
			description:   "Should find record 2 (has both 'medieval' and 'character' in creator_notes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := resource.Filter{
				ContentFilters: []resource.ContentFilter{tt.filter},
			}

			count, err := store.Count(filter)
			require.NoError(t, err, "Count should not error for %s", tt.name)
			assert.Equal(t, tt.expectedCount, count, "%s: %s", tt.name, tt.description)
		})
	}
}
