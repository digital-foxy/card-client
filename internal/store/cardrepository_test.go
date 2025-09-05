package store

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/r3dpixel/card-client/internal/ent/card"
	"github.com/r3dpixel/card-client/internal/ent/schema"
	"github.com/r3dpixel/card-client/services/filter"
	"github.com/r3dpixel/toolkit/slicesx"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/internal/ent/enttest"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, dialect.SQLite, dsn)
	return client
}

func seedTags(t *testing.T, ctx context.Context, client *ent.Client, tags ...models.Tag) {
	t.Helper()
	builders := make([]*ent.TagCreate, len(tags))
	for i, tag := range tags {
		builders[i] = client.Tag.Create().SetID(scheme.TagID(tag.Slug)).SetName(tag.Name)
	}
	_, err := client.Tag.CreateBulk(builders...).Save(ctx)
	require.NoError(t, err)
}

func seedCard(t *testing.T, ctx context.Context, client *ent.Client, metadata *models.Metadata) *ent.Card {
	t.Helper()
	tagIDs := make([]scheme.TagID, len(metadata.Tags))
	for i, tag := range metadata.Tags {
		tagIDs[i] = scheme.TagID(tag.Slug)
	}
	metadata.DirectURL = metadata.CardURL

	card, err := client.Card.Create().
		SetSource(metadata.Source).
		SetCardURL(metadata.CardURL).
		SetDirectURL(metadata.DirectURL).
		SetPlatformID(metadata.PlatformID).
		SetCharacterID(metadata.CharacterID).
		SetCardName(metadata.CardName).
		SetCharacterName(metadata.CharacterName).
		SetCreator(metadata.Creator).
		SetTagline(metadata.Tagline).
		SetCreateTime(metadata.CreateTime).
		SetUpdateTime(metadata.UpdateTime).
		SetBookUpdateTime(metadata.BookUpdateTime).
		SetImportTime(timestamp.Now[timestamp.Nano]()).
		SetCheckTime(timestamp.Now[timestamp.Nano]()).
		SetLastUpdateStatus(scheme.UpdateSuccess).
		AddTagIDs(tagIDs...).
		Save(ctx)
	require.NoError(t, err)
	return card
}

func TestCardRepository_Count_Lifecycle(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	initialCount := repo.count(client, ctx)
	assert.Equal(t, 0, initialCount, "Count should be 0 for an empty database")

	card1 := seedCard(t, ctx, client, &models.Metadata{Source: source.WyvernChat, CardURL: "url1"})
	seedCard(t, ctx, client, &models.Metadata{Source: source.ChubAI, CardURL: "url2"})

	afterSeedCount := repo.count(client, ctx)
	assert.Equal(t, 2, afterSeedCount, "Count should be 2 after seeding two cards")

	err := client.Card.DeleteOne(card1).Exec(ctx)
	require.NoError(t, err)

	afterDeleteCount := repo.count(client, ctx)
	assert.Equal(t, 1, afterDeleteCount, "Count should be 1 after deleting one card")
}

func TestCardRepository_DataIntegrity(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	testMetadata := &models.Metadata{
		Source:        source.ChubAI,
		CardURL:       "https://chub.ai/characters/test-char",
		CardName:      "Test Card Name",
		CharacterName: "Test Character",
		Creator:       "A. Creator",
		UpdateTime:    1678886400000,
	}

	seededCard := seedCard(t, ctx, client, testMetadata)
	require.NotNil(t, seededCard, "Seeded card should not be nil")

	retrievedCard, err := client.Card.Query().
		Where(card.IDEQ(seededCard.ID)).
		Only(ctx)

	require.NoError(t, err, "Should be able to retrieve the seeded card")
	require.NotNil(t, retrievedCard, "Retrieved card should not be nil")

	assert.Equal(t, testMetadata.Source, retrievedCard.Source, "Source field should match")
	assert.Equal(t, testMetadata.CardURL, retrievedCard.CardURL, "CardURL field should match")
	assert.Equal(t, testMetadata.CardName, retrievedCard.CardName, "CardName field should match")
	assert.Equal(t, testMetadata.CharacterName, retrievedCard.CharacterName, "CharacterName field should match")
	assert.Equal(t, testMetadata.Creator, retrievedCard.Creator, "Creator field should match")

	assert.Equal(t, testMetadata.UpdateTime, retrievedCard.UpdateTime, "UpdateTime should match")

	assert.NotZero(t, retrievedCard.ImportTime, "ImportTime should be set")
	assert.NotZero(t, retrievedCard.CheckTime, "CheckTime should be set")
	assert.Equal(t, scheme.UpdateSuccess, retrievedCard.LastUpdateStatus, "LastUpdateStatus should be success on initial import")
}

func TestCardRepository_InsertCard(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	tags := []models.Tag{{Slug: "tag1", Name: "Tag One"}, {Slug: "tag2", Name: "Tag Two"}}
	seedTags(t, ctx, client, tags...)

	metadata := &models.Metadata{
		Source:         source.ChubAI,
		CardURL:        "https://chub.ai/characters/test",
		DirectURL:      "https://chub.ai/characters/test",
		PlatformID:     "pid-1",
		CharacterID:    "cid-1",
		CardName:       "Test Card",
		CharacterName:  "Test Character",
		Creator:        "Test Creator",
		Tagline:        "A test tagline",
		CreateTime:     1000,
		UpdateTime:     2000,
		BookUpdateTime: 3000,
		Tags:           tags,
	}
	importTime := timestamp.Now[timestamp.Nano]()
	tagIDs := slicesx.Map(metadata.Tags, func(tag models.Tag) scheme.TagID {
		return scheme.TagID(tag.Slug)
	})

	header, err := repo.insertCard(client, ctx, metadata, tagIDs, importTime, 0)
	require.NoError(t, err)
	assert.NotNil(t, header)
	assert.Equal(t, metadata.CardURL, header.CardURL)
	assert.Equal(t, metadata.Creator, header.Creator)
	assert.Len(t, header.Tags, 2)

	t.Run("fails on duplicate CardURL", func(t *testing.T) {
		tagIDs := slicesx.Map(metadata.Tags, func(tag models.Tag) scheme.TagID {
			return scheme.TagID(tag.Slug)
		})
		_, err := repo.insertCard(client, ctx, metadata, tagIDs, importTime, 0)
		require.Error(t, err)
		assert.True(t, ent.IsConstraintError(err))
	})
}

func TestCardRepository_FindAllCards(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	tags := []models.Tag{{Slug: "a", Name: "A"}, {Slug: "b", Name: "B"}}
	seedTags(t, ctx, client, tags...)
	seedCard(t, ctx, client, &models.Metadata{Source: source.WyvernChat, CardURL: "url1", Tags: tags})
	seedCard(t, ctx, client, &models.Metadata{Source: source.ChubAI, CardURL: "url2"})
	seedCard(t, ctx, client, &models.Metadata{Source: source.Pygmalion, CardURL: "url3"})

	t.Run("gets all cards", func(t *testing.T) {
		cardIDs := repo.findPagedCardIDs(client, ctx, filter.SearchFilter{}, 0, -1)
		assert.Len(t, cardIDs, 3)
	})

	t.Run("respects pagination with ordering", func(t *testing.T) {
		allIDs := repo.findPagedCardIDs(client, ctx, filter.SearchFilter{}, -1, -1)
		require.Len(t, allIDs, 3, "Test setup should have 3 cards")

		page1IDs := repo.findPagedCardIDs(client, ctx, filter.SearchFilter{}, 0, 1)
		page2IDs := repo.findPagedCardIDs(client, ctx, filter.SearchFilter{}, 1, 1)
		page3IDs := repo.findPagedCardIDs(client, ctx, filter.SearchFilter{}, 2, 1)

		require.Len(t, page1IDs, 1)
		assert.Equal(t, allIDs[0], page1IDs[0], "First page should contain the first card ID")

		require.Len(t, page2IDs, 1)
		assert.Equal(t, allIDs[1], page2IDs[0], "Second page should contain the second card ID")

		require.Len(t, page3IDs, 1)
		assert.Equal(t, allIDs[2], page3IDs[0], "Third page should contain the third card ID")
	})

	t.Run("returns empty slice for no results", func(t *testing.T) {
		cardIDs := repo.findPagedCardIDs(client, ctx, filter.SearchFilter{}, 10, 5)
		assert.Len(t, cardIDs, 0)
	})
}

func TestCardRepository_Finders(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	c1 := seedCard(t, ctx, client, &models.Metadata{Source: source.Pygmalion, CardURL: "url1.com", UpdateTime: 1000, BookUpdateTime: 1500, Creator: "c1"})
	c2 := seedCard(t, ctx, client, &models.Metadata{Source: source.WyvernChat, CardURL: "url2.com", UpdateTime: 2000, BookUpdateTime: 2500, Creator: "c2"})
	seedCard(t, ctx, client, &models.Metadata{Source: source.ChubAI, CardURL: "url3.com", UpdateTime: 3000, BookUpdateTime: 3500, Creator: "c3"})

	t.Run("findURLs", func(t *testing.T) {
		urls := repo.findURLs(client, ctx, []string{"url1.com", "url-non-existent.com", "url3.com"})
		assert.ElementsMatch(t, []string{"url1.com", "url3.com"}, urls)
	})

	t.Run("findMiniHeaders", func(t *testing.T) {
		headersSlice := repo.findMiniHeaders(client, ctx, []scheme.CardID{c1.ID, c2.ID})
		require.Len(t, headersSlice, 2)

		headersByID := make(map[scheme.CardID]scheme.MiniHeader)
		for _, h := range headersSlice {
			headersByID[h.CardID] = h
		}

		assert.Contains(t, headersByID, c1.ID)
		assert.Contains(t, headersByID, c2.ID)
		assert.Equal(t, "url1.com", headersByID[c1.ID].CardURL)
		assert.Equal(t, timestamp.Nano(1500), headersByID[c1.ID].BookUpdateTime)
		assert.Equal(t, "c2", headersByID[c2.ID].Creator)
	})

	t.Run("findMiscHeaders", func(t *testing.T) {
		headersSlice := repo.findMiscHeaders(client, ctx, []scheme.CardID{c1.ID, c2.ID})
		require.Len(t, headersSlice, 2)

		headersByID := make(map[scheme.CardID]scheme.MiscHeader)
		for _, h := range headersSlice {
			headersByID[h.CardID] = h
		}

		assert.Contains(t, headersByID, c1.ID)
		assert.Contains(t, headersByID, c2.ID)
		assert.Equal(t, timestamp.Nano(1000), headersByID[c1.ID].UpdateTime)
	})
}

func TestCardRepository_FindSingleHeaders(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	metadata := &models.Metadata{
		CardURL:        "single-header-url.com",
		Creator:        "a-creator",
		UpdateTime:     12345,
		BookUpdateTime: 67890,
		Source:         source.ChubAI,
		CardName:       "Test Card Name",
		CharacterName:  "Test Character Name",
		PlatformID:     "pid-single",
		CharacterID:    "cid-single",
	}
	seededCard := seedCard(t, ctx, client, metadata)

	t.Run("findMiniHeader", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			header, err := repo.findMiniHeader(client, ctx, seededCard.ID)
			require.NoError(t, err)

			assert.Equal(t, seededCard.ID, header.CardID)
			assert.Equal(t, "single-header-url.com", header.CardURL)
			assert.Equal(t, "a-creator", header.Creator)
			assert.Equal(t, timestamp.Nano(12345), header.UpdateTime)
			assert.Equal(t, timestamp.Nano(67890), header.BookUpdateTime)
		})

		t.Run("not found", func(t *testing.T) {
			_, err := repo.findMiniHeader(client, ctx, scheme.CardID(uuid.NewString()))
			require.Error(t, err)
		})
	})

	t.Run("findMiscHeader", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			header, err := repo.findMiscHeader(client, ctx, seededCard.ID)
			require.NoError(t, err)

			assert.Equal(t, seededCard.ID, header.CardID)
			assert.Equal(t, source.ChubAI, header.Source)
			assert.Equal(t, "pid-single", header.PlatformID)
			assert.Equal(t, "cid-single", header.CharacterID)
			assert.Equal(t, "Test Card Name", header.CardName)
			assert.Equal(t, "Test Character Name", header.CharacterName)
			assert.Equal(t, "a-creator", header.Creator)
			assert.Equal(t, timestamp.Nano(12345), header.UpdateTime)
		})

		t.Run("not found", func(t *testing.T) {
			_, err := repo.findMiscHeader(client, ctx, scheme.CardID(uuid.NewString()))
			require.Error(t, err)
		})
	})
}

func TestCardRepository_FindSpecificCards(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	tags := []models.Tag{{Slug: "tag1", Name: "Tag One"}, {Slug: "tag2", Name: "Tag Two"}}
	seedTags(t, ctx, client, tags...)

	c1 := seedCard(t, ctx, client, &models.Metadata{CardURL: "url1", Creator: "Creator1", Source: source.NyaiMe, CharacterID: "1", PlatformID: "1", Tags: tags})
	c2 := seedCard(t, ctx, client, &models.Metadata{CardURL: "url2", Creator: "Creator2", Source: source.ChubAI, CharacterID: "2", PlatformID: "2", Tags: tags})
	c3 := seedCard(t, ctx, client, &models.Metadata{CardURL: "url3", Creator: "Creator3", Source: source.Pygmalion, CharacterID: "3", PlatformID: "3"})

	exportData1 := scheme.ExportHeader{ExportTime: 1111, LastExportedVersion: 1000}
	err := repo.updateCardExportData(client, ctx, c1.ID, exportData1)
	require.NoError(t, err)

	exportData3 := scheme.ExportHeader{ExportTime: 3333, LastExportedVersion: 3000}
	err = repo.updateCardExportData(client, ctx, c3.ID, exportData3)
	require.NoError(t, err)

	t.Run("findCards", func(t *testing.T) {
		t.Run("finds multiple existing cards", func(t *testing.T) {
			headers, readAt := repo.findCards(client, ctx, []scheme.CardID{c1.ID, c3.ID})
			assert.NotZero(t, readAt)
			require.Len(t, headers, 2)

			headersByID := make(map[scheme.CardID]scheme.CardHeader)
			for _, h := range headers {
				headersByID[h.CardID] = h
			}

			assert.Contains(t, headersByID, c1.ID)
			assert.Equal(t, "url1", headersByID[c1.ID].CardURL)
			require.Len(t, headersByID[c1.ID].Tags, 2)
			assert.Equal(t, scheme.TagID("tag1"), headersByID[c1.ID].Tags[0].ID)
			assert.Equal(t, scheme.TagID("tag2"), headersByID[c1.ID].Tags[1].ID)

			assert.Contains(t, headersByID, c3.ID)
			assert.Equal(t, "url3", headersByID[c3.ID].CardURL)
			assert.Empty(t, headersByID[c3.ID].Tags)
		})

		t.Run("handles a mix of existing and non-existing cards", func(t *testing.T) {
			nonExistentID := scheme.CardID(uuid.NewString())
			headers, _ := repo.findCards(client, ctx, []scheme.CardID{c2.ID, nonExistentID})
			require.Len(t, headers, 1)
			assert.Equal(t, c2.ID, headers[0].CardID)
		})

		t.Run("returns an empty slice for no matching cards", func(t *testing.T) {
			headers, _ := repo.findCards(client, ctx, []scheme.CardID{scheme.CardID(uuid.NewString())})
			assert.Empty(t, headers)
		})

		t.Run("returns an empty slice for empty input", func(t *testing.T) {
			headers, _ := repo.findCards(client, ctx, []scheme.CardID{})
			assert.Empty(t, headers)
		})
	})

	t.Run("findExportPayloads", func(t *testing.T) {
		t.Run("finds payloads for multiple existing cards", func(t *testing.T) {
			payloads, readAt := repo.findExportPayloads(client, ctx, []scheme.CardID{c1.ID, c2.ID, scheme.CardID(uuid.NewString())})

			assert.NotZero(t, readAt)
			require.Len(t, payloads, 2)

			payloadsByID := make(map[scheme.CardID]scheme.IdExportHeader)
			for _, p := range payloads {
				payloadsByID[p.CardID] = p
			}

			assert.Contains(t, payloadsByID, c1.ID)
			assert.Equal(t, exportData1.ExportTime, payloadsByID[c1.ID].ExportTime)
			assert.Equal(t, exportData1.LastExportedVersion, payloadsByID[c1.ID].LastExportedVersion)

			assert.Contains(t, payloadsByID, c2.ID)
			assert.Zero(t, payloadsByID[c2.ID].ExportTime)
			assert.Zero(t, payloadsByID[c2.ID].LastExportedVersion)
		})

		t.Run("returns an empty slice for empty input", func(t *testing.T) {
			payloads, _ := repo.findExportPayloads(client, ctx, []scheme.CardID{})
			assert.Empty(t, payloads)
		})
	})
}

func TestCardRepository_UpdateCard(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	tagsOld := []models.Tag{{Slug: "old-tag1", Name: "Old1"}, {Slug: "old-tag2", Name: "Old2"}}
	tagsNew := []models.Tag{{Slug: "new-tag1", Name: "New1"}, {Slug: "new-tag2", Name: "New2"}}
	seedTags(t, ctx, client, tagsOld...)
	seedTags(t, ctx, client, tagsNew...)

	seededCard := seedCard(t, ctx, client, &models.Metadata{Source: source.ChubAI, CardURL: "url-orig", Tags: tagsOld})
	oldCard, err := client.Card.Query().Where(card.IDEQ(seededCard.ID)).WithTags().Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "url-orig", oldCard.CardURL)
	assert.Len(t, oldCard.Edges.Tags, 2)
	assert.Equal(t, "old-tag1", string(oldCard.Edges.Tags[0].ID))
	assert.Equal(t, "old-tag2", string(oldCard.Edges.Tags[1].ID))

	newMetadata := &models.Metadata{
		Source:  source.ChubAI,
		CardURL: "url-new",
		Tags:    tagsNew,
	}
	updateHeader := scheme.UpdateHeader{CheckTime: 9999}
	tagIDs := slicesx.Map(newMetadata.Tags, func(tag models.Tag) scheme.TagID {
		return scheme.TagID(tag.Slug)
	})
	header, err := repo.updateCard(client, ctx, seededCard.ID, newMetadata, tagIDs, updateHeader)
	require.NoError(t, err)

	assert.Equal(t, "url-new", header.CardURL)
	assert.Len(t, header.Tags, 2)
	assert.Equal(t, "new-tag1", string(header.Tags[0].ID))
	assert.Equal(t, "new-tag2", string(header.Tags[1].ID))

	updatedCard, err := client.Card.Query().Where(card.IDEQ(seededCard.ID)).WithTags().Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "url-new", updatedCard.CardURL)
	assert.Equal(t, timestamp.Nano(9999), updatedCard.CheckTime)
	assert.Len(t, updatedCard.Edges.Tags, 2)
	assert.Equal(t, "new-tag1", string(updatedCard.Edges.Tags[0].ID))
	assert.Equal(t, "new-tag2", string(updatedCard.Edges.Tags[1].ID))
}

func TestCardRepository_UpdateCardStatus(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	card := seedCard(t, ctx, client, &models.Metadata{Source: source.ChubAI, CardURL: "url"})

	err := repo.updateCardStatus(client, ctx, card.ID, 8888, scheme.UpdateFailed)
	require.NoError(t, err)

	updatedCard, err := client.Card.Get(ctx, card.ID)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(8888), updatedCard.CheckTime)
	assert.Equal(t, scheme.UpdateFailed, updatedCard.LastUpdateStatus)
}

func TestCardRepository_UpdateCardExportData(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}
	card := seedCard(t, ctx, client, &models.Metadata{Source: source.PepHop, CardURL: "url"})

	exportData := scheme.ExportHeader{
		ExportTime:          7777,
		LastExportedVersion: 6666,
	}
	err := repo.updateCardExportData(client, ctx, card.ID, exportData)
	require.NoError(t, err)

	updatedCard, err := client.Card.Get(ctx, card.ID)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(7777), updatedCard.ExportTime)
	assert.Equal(t, timestamp.Nano(6666), updatedCard.LastExportedVersion)
}

func TestCardRepository_UpdateToLatestExport(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	c1 := seedCard(t, ctx, client, &models.Metadata{Source: source.NyaiMe, CardURL: "url1", UpdateTime: 1000})
	c2 := seedCard(t, ctx, client, &models.Metadata{Source: source.RisuAI, CardURL: "url2", UpdateTime: 2000})
	c3 := seedCard(t, ctx, client, &models.Metadata{Source: source.CharacterTavern, CardURL: "url3", UpdateTime: 3000})

	err := repo.updateToLatestExport(client, ctx, c1.ID, 5000)
	require.NoError(t, err)

	err = repo.updateToLatestExport(client, ctx, c2.ID, 5000)
	require.NoError(t, err)

	updatedC1, err := client.Card.Get(ctx, c1.ID)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(5000), updatedC1.ExportTime)
	assert.Equal(t, timestamp.Nano(1000), updatedC1.LastExportedVersion)

	updatedC2, err := client.Card.Get(ctx, c2.ID)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(5000), updatedC2.ExportTime)
	assert.Equal(t, timestamp.Nano(2000), updatedC2.LastExportedVersion)

	notUpdatedC3, err := client.Card.Get(ctx, c3.ID)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(0), notUpdatedC3.ExportTime)
	assert.Equal(t, timestamp.Nano(0), notUpdatedC3.LastExportedVersion)
}

func TestStructTagsMatchSchemaConstants(t *testing.T) {
	t.Run("MiniHeader struct tags match schema constants", func(t *testing.T) {
		expectedTags := map[string]string{
			"CardID":         schema.FieldCardID,
			"CardURL":        schema.FieldCardURL,
			"Creator":        schema.FieldCardCreator,
			"UpdateTime":     schema.FieldCardUpdateTime,
			"BookUpdateTime": schema.FieldCardBookUpdateTime,
		}

		structType := reflect.TypeOf(scheme.MiniHeader{})
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			sqlTag := field.Tag.Get("sql")
			expected, ok := expectedTags[field.Name]

			assert.True(t, ok, "No expected tag found for field %s", field.Name)
			assert.Equal(t, expected, sqlTag, "Tag mismatch for field %s", field.Name)
		}
	})

	t.Run("MiscHeader struct tags match schema constants", func(t *testing.T) {
		expectedTags := map[string]string{
			"CardID":        schema.FieldCardID,
			"Source":        schema.FieldCardSource,
			"PlatformID":    schema.FieldCardPlatformID,
			"CharacterID":   schema.FieldCardCharacterID,
			"CardName":      schema.FieldCardName,
			"CharacterName": schema.FieldCardCharacterName,
			"Creator":       schema.FieldCardCreator,
			"UpdateTime":    schema.FieldCardUpdateTime,
		}

		structType := reflect.TypeOf(scheme.MiscHeader{})
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			sqlTag := field.Tag.Get("sql")
			expected, ok := expectedTags[field.Name]

			assert.True(t, ok, "No expected tag found for field %s", field.Name)
			assert.Equal(t, expected, sqlTag, "Tag mismatch for field %s", field.Name)
		}
	})
}

func TestCardRepository_FavoriteFunctions(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &cardRepository{}

	c1 := seedCard(t, ctx, client, &models.Metadata{Source: source.ChubAI, CardURL: "fav_url1", CharacterID: "1", PlatformID: "1"})
	c2 := seedCard(t, ctx, client, &models.Metadata{Source: source.Pygmalion, CardURL: "fav_url2", CharacterID: "2", PlatformID: "2"})
	c3 := seedCard(t, ctx, client, &models.Metadata{Source: source.NyaiMe, CardURL: "fav_url3", CharacterID: "3", PlatformID: "3"})

	t.Run("toggleFavorite", func(t *testing.T) {
		initialCard, err := client.Card.Get(ctx, c1.ID)
		require.NoError(t, err)
		assert.False(t, initialCard.Favorite, "Card should initially not be a favorite")

		err = repo.toggleFavorite(client, ctx, c1.ID)
		require.NoError(t, err)

		toggledCard, err := client.Card.Get(ctx, c1.ID)
		require.NoError(t, err)
		assert.True(t, toggledCard.Favorite, "Card should be a favorite after first toggle")

		err = repo.toggleFavorite(client, ctx, c1.ID)
		require.NoError(t, err)

		toggledBackCard, err := client.Card.Get(ctx, c1.ID)
		require.NoError(t, err)
		assert.False(t, toggledBackCard.Favorite, "Card should not be a favorite after second toggle")
	})

	t.Run("setFavorites", func(t *testing.T) {
		err := repo.setFavorites(client, ctx, []scheme.CardID{c2.ID, c3.ID}, true)
		require.NoError(t, err)

		updatedC2, err := client.Card.Get(ctx, c2.ID)
		require.NoError(t, err)
		assert.True(t, updatedC2.Favorite)

		updatedC3, err := client.Card.Get(ctx, c3.ID)
		require.NoError(t, err)
		assert.True(t, updatedC3.Favorite)

		unaffectedC1, err := client.Card.Get(ctx, c1.ID)
		require.NoError(t, err)
		assert.False(t, unaffectedC1.Favorite)

		err = repo.setFavorites(client, ctx, []scheme.CardID{c2.ID}, false)
		require.NoError(t, err)

		finalC2, err := client.Card.Get(ctx, c2.ID)
		require.NoError(t, err)
		assert.False(t, finalC2.Favorite)

		finalC3, err := client.Card.Get(ctx, c3.ID)
		require.NoError(t, err)
		assert.True(t, finalC3.Favorite)
	})
}
