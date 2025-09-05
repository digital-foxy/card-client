package store

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/r3dpixel/card-client/services/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/internal/ent/enttest"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-fetcher/source"
	p "github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

func setupIntegrationTest(t *testing.T) (*Service, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	pngOpts := opts.PngOptions{MaxVersions: 5, ThumbnailSize: 128}
	s := NewService(client, "testVault", t.TempDir(), pngOpts)
	return s, client
}

func createTestEditableCard(t *testing.T) *p.CharacterCard {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	img.Set(5, 5, color.RGBA{255, 0, 0, 255})
	buf := new(bytes.Buffer)
	require.NoError(t, png.Encode(buf, img))
	tmpFile := filepath.Join(t.TempDir(), "card.png")
	require.NoError(t, os.WriteFile(tmpFile, buf.Bytes(), 0644))
	rawCard, err := p.FromFile(tmpFile).Get()
	require.NoError(t, err)
	characterCard, err := rawCard.Decode()
	require.NoError(t, err)
	return characterCard
}

func TestService_Count_Integration(t *testing.T) {
	s, _ := setupIntegrationTest(t)
	ctx := context.Background()

	initialCount := s.Count(ctx)
	assert.Equal(t, 0, initialCount, "Count should be 0 for an empty database")

	rawCard := createTestEditableCard(t)
	_, err := s.InsertCard(ctx, &models.Metadata{CardURL: "http://example.com/card1", DirectURL: "example.com/c1", Source: source.ChubAI, PlatformID: "1", CharacterID: "1"}, rawCard, 1, 0)
	require.NoError(t, err)
	_, err = s.InsertCard(ctx, &models.Metadata{CardURL: "http://example.com/card2", DirectURL: "example.com/c2", Source: source.ChubAI, PlatformID: "2", CharacterID: "2"}, rawCard, 2, 0)
	require.NoError(t, err)

	finalCount := s.Count(ctx)
	assert.Equal(t, 2, finalCount, "Count should be 2 after inserting two cards")
}

func TestService_InsertCard_Integration(t *testing.T) {
	rawCard := createTestEditableCard(t)
	metadata := &models.Metadata{
		CardURL: "http://example.com/card1",
		Source:  source.ChubAI,
		Tags:    []models.Tag{{Slug: "tag-a", Name: "Tag A"}},
	}
	importTime := timestamp.Now[timestamp.Nano]()

	t.Run("success path", func(t *testing.T) {
		s, client := setupIntegrationTest(t)
		ctx := context.Background()

		header, err := s.InsertCard(context.Background(), metadata, rawCard, importTime, 0)
		require.NoError(t, err)
		require.NotNil(t, header)

		dbCard, err := client.Card.Get(ctx, header.CardID)
		require.NoError(t, err)
		assert.Equal(t, metadata.CardURL, dbCard.CardURL)

		dbTags, err := dbCard.QueryTags().All(ctx)
		require.NoError(t, err)
		assert.Len(t, dbTags, 1)
		assert.Equal(t, "tag-a", string(dbTags[0].ID))

		pngPath := s.GetPngPath(header.CardID, header.UpdateTime)
		_, err = os.Stat(pngPath)
		assert.NoError(t, err, "png file should exist on disk")
	})

	t.Run("transaction rolls back on card insert failure", func(t *testing.T) {
		s, client := setupIntegrationTest(t)
		ctx := context.Background()

		_, err := s.InsertCard(context.Background(), metadata, rawCard, importTime, 0)
		require.NoError(t, err)

		_, err = s.InsertCard(context.Background(), metadata, rawCard, importTime, 1)
		require.Error(t, err)
		assert.True(t, ent.IsConstraintError(err))

		tagCount, err := client.Tag.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, tagCount, "no new tags should be committed on failure")
	})

	t.Run("transaction rolls back on png save failure", func(t *testing.T) {
		s, client := setupIntegrationTest(t)
		ctx := context.Background()

		s.pngRepository.rootDir = "/a-non-writable-dir"

		_, err := s.InsertCard(context.Background(), metadata, rawCard, importTime, 0)
		require.Error(t, err)

		cardCount, err := client.Card.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, cardCount, "card should not be committed on failure")

		tagCount, err := client.Tag.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, tagCount, "tags should not be committed on failure")
	})
}

func TestService_UpdateCard_Integration(t *testing.T) {
	s, client := setupIntegrationTest(t)
	ctx := context.Background()
	rawCard := createTestEditableCard(t)
	checkTime := timestamp.Now[timestamp.Nano]()
	initialMetadata := &models.Metadata{
		CardURL:    "http://example.com/card-to-update",
		UpdateTime: 1000,
		Source:     source.ChubAI,
		Tags:       []models.Tag{{Slug: "tag-a", Name: "Tag A"}},
	}
	header, err := s.InsertCard(context.Background(), initialMetadata, rawCard, 1, 0)
	require.NoError(t, err)

	updatedMetadata := &models.Metadata{
		CardURL:    "http://example.com/card-updated",
		UpdateTime: 2000,
		Source:     source.ChubAI,
		Tags:       []models.Tag{{Slug: "tag-b", Name: "Tag B"}},
	}
	newRawCard := createTestEditableCard(t)

	updatedHeader, err := s.UpdateCard(context.Background(), header.CardID, updatedMetadata, newRawCard, checkTime)
	require.NoError(t, err)
	assert.Equal(t, timestamp.Nano(2000), updatedHeader.UpdateTime)

	dbCard, err := client.Card.Get(ctx, header.CardID)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/card-updated", dbCard.CardURL)
	assert.Equal(t, checkTime, dbCard.CheckTime)

	tags, err := dbCard.QueryTags().All(ctx)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, "tag-b", string(tags[0].ID))

	newPngPath := s.GetPngPath(header.CardID, 2000)
	_, err = os.Stat(newPngPath)
	assert.NoError(t, err, "new png version should exist on disk")
}

func TestService_Finders_Integration(t *testing.T) {
	s, _ := setupIntegrationTest(t)
	rawCard := createTestEditableCard(t)
	h1, _ := s.InsertCard(context.Background(), &models.Metadata{CardURL: "url1", DirectURL: "1", Source: source.ChubAI, PlatformID: "f1", CharacterID: "f1", UpdateTime: 1000}, rawCard, 1, 0)
	h2, _ := s.InsertCard(context.Background(), &models.Metadata{CardURL: "url2", DirectURL: "2", Source: source.ChubAI, PlatformID: "f2", CharacterID: "f2", UpdateTime: 1000}, rawCard, 2, 0)
	_, _ = s.InsertCard(context.Background(), &models.Metadata{CardURL: "url1", DirectURL: "3", Source: source.ChubAI, PlatformID: "f3", CharacterID: "f3"}, rawCard, 2, 1)
	_, _ = s.InsertCard(context.Background(), &models.Metadata{CardURL: "url5", DirectURL: "4", Source: source.ChubAI, PlatformID: "f1", CharacterID: "f5"}, rawCard, 2, 2)
	_, _ = s.InsertCard(context.Background(), &models.Metadata{CardURL: "url6", DirectURL: "5", Source: source.ChubAI, PlatformID: "f6", CharacterID: "f1"}, rawCard, 2, 3)
	ctx := context.Background()
	t.Run("FindPagedIDs", func(t *testing.T) {
		cards := s.FindPagedIDs(ctx, filter.SearchFilter{}, 0, 10)
		assert.Len(t, cards, 2)
	})

	t.Run("FindURLs", func(t *testing.T) {
		urls := s.FindURLs(ctx, []string{"url1", "url-non-existent"})
		assert.ElementsMatch(t, []string{"url1"}, urls)
	})

	t.Run("FindMiniHeaders", func(t *testing.T) {
		cardIDs := []scheme.CardID{h1.CardID, h2.CardID}
		headers := s.FindMiniHeaders(ctx, cardIDs)
		assert.Len(t, headers, 2)
		for _, h := range headers {
			if h.CardID == h1.CardID {
				assert.Equal(t, timestamp.Nano(1000), h.UpdateTime)
			}
		}
	})

	t.Run("FindMiscHeaders", func(t *testing.T) {
		cardIDs := []scheme.CardID{h1.CardID, h2.CardID}
		headers := s.FindMiscHeaders(ctx, cardIDs)
		assert.Len(t, headers, 2)
		for _, h := range headers {
			if h.CardID == h2.CardID {
				assert.Equal(t, source.ChubAI, h.Source)
			}
		}
	})
}

func TestService_FindSingleHeader_Integration(t *testing.T) {
	s, _ := setupIntegrationTest(t)
	ctx := context.Background()
	rawCard := createTestEditableCard(t)

	metadata := &models.Metadata{
		CardURL:        "find-me.com",
		Source:         source.ChubAI,
		Creator:        "test-creator",
		UpdateTime:     5000,
		BookUpdateTime: 5500,
	}
	seededHeader, err := s.InsertCard(ctx, metadata, rawCard, 1, 0)
	require.NoError(t, err)

	t.Run("FindMiniHeader", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			header, err := s.FindMiniHeader(ctx, seededHeader.CardID)
			require.NoError(t, err, "Should not error when card exists")

			assert.Equal(t, seededHeader.CardID, header.CardID)
			assert.Equal(t, "find-me.com", header.CardURL)
			assert.Equal(t, timestamp.Nano(5000), header.UpdateTime)
			assert.Equal(t, timestamp.Nano(5500), header.BookUpdateTime)
		})

		t.Run("not found", func(t *testing.T) {
			nonExistentID := scheme.CardID(uuid.NewString())
			_, err := s.FindMiniHeader(ctx, nonExistentID)

			require.Error(t, err, "Should error when card does not exist")
		})
	})

	t.Run("FindMiscHeader", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			header, err := s.FindMiscHeader(ctx, seededHeader.CardID)
			require.NoError(t, err, "Should not error when card exists")

			assert.Equal(t, seededHeader.CardID, header.CardID)
			assert.Equal(t, source.ChubAI, header.Source)
			assert.Equal(t, "test-creator", header.Creator)
		})

		t.Run("not found", func(t *testing.T) {
			nonExistentID := scheme.CardID(uuid.NewString())
			_, err := s.FindMiscHeader(ctx, nonExistentID)

			require.Error(t, err, "Should error when card does not exist")
		})
	})
}

func TestService_FindSpecific_Integration(t *testing.T) {
	s, _ := setupIntegrationTest(t)
	ctx := context.Background()
	rawCard := createTestEditableCard(t)

	tags := []models.Tag{{Slug: "fic", Name: "Fictional Character"}}
	h1, err := s.InsertCard(ctx, &models.Metadata{CardURL: "url1", DirectURL: "1", Source: source.ChubAI, CharacterID: "1", PlatformID: "1", Tags: tags, UpdateTime: 2}, rawCard, 1, 0)
	require.NoError(t, err)

	h2, err := s.InsertCard(ctx, &models.Metadata{CardURL: "url2", DirectURL: "2", Source: source.Pygmalion, CharacterID: "2", PlatformID: "2"}, rawCard, 2, 1)
	require.NoError(t, err)

	h3, err := s.InsertCard(ctx, &models.Metadata{CardURL: "url3", DirectURL: "3", Source: source.WyvernChat, CharacterID: "3", PlatformID: "3"}, rawCard, 3, 2)

	// Add export data to some cards
	exportTime := timestamp.Now[timestamp.Nano]()

	err = s.UpdateToLatestExport(ctx, h1.CardID, exportTime)
	require.NoError(t, err)
	err = s.UpdateToLatestExport(ctx, h3.CardID, exportTime)
	require.NoError(t, err)

	t.Run("FindPagedIDs", func(t *testing.T) {
		t.Run("successfully finds a list of cards by ResourceID", func(t *testing.T) {
			cardIDs := []scheme.CardID{h1.CardID, h3.CardID}
			headers, readAt := s.FindCards(ctx, cardIDs)

			assert.NotZero(t, readAt)
			require.Len(t, headers, 2)

			headerMap := make(map[scheme.CardID]scheme.CardHeader)
			for _, h := range headers {
				headerMap[h.CardID] = h
			}

			assert.Equal(t, source.ChubAI, headerMap[h1.CardID].Source)
			assert.Len(t, headerMap[h1.CardID].Tags, 1)
			assert.Equal(t, source.WyvernChat, headerMap[h3.CardID].Source)
			assert.Empty(t, headerMap[h3.CardID].Tags)
		})

		t.Run("returns only found cards when some IDs don't exist", func(t *testing.T) {
			nonExistentID := scheme.CardID(uuid.NewString())
			cardIDs := []scheme.CardID{h2.CardID, nonExistentID}
			headers, _ := s.FindCards(ctx, cardIDs)

			require.Len(t, headers, 1)
			assert.Equal(t, h2.CardID, headers[0].CardID)
		})

		t.Run("returns an empty slice for an empty list of IDs", func(t *testing.T) {
			headers, _ := s.FindCards(ctx, []scheme.CardID{})
			assert.Empty(t, headers)
		})
	})

	t.Run("FlushExportPayloads", func(t *testing.T) {
		t.Run("successfully finds export payloads by ResourceID", func(t *testing.T) {
			cardIDs := []scheme.CardID{h1.CardID, h2.CardID, scheme.CardID(uuid.NewString())}
			payloads, readAt := s.FindIdExportHeaders(ctx, cardIDs)

			assert.NotZero(t, readAt)
			require.Len(t, payloads, 2)

			payloadMap := make(map[scheme.CardID]scheme.IdExportHeader)
			for _, payload := range payloads {
				payloadMap[payload.CardID] = payload
			}

			assert.Equal(t, exportTime, payloadMap[h1.CardID].ExportTime)
			assert.NotZero(t, payloadMap[h1.CardID].LastExportedVersion)

			assert.Zero(t, payloadMap[h2.CardID].ExportTime)
			assert.Zero(t, payloadMap[h2.CardID].LastExportedVersion)
		})

		t.Run("returns an empty slice for an empty list of IDs", func(t *testing.T) {
			payloads, _ := s.FindIdExportHeaders(ctx, []scheme.CardID{})
			assert.Empty(t, payloads)
		})
	})
}

func TestService_UpdateToLatestExport_Integration(t *testing.T) {
	s, client := setupIntegrationTest(t)
	ctx := context.Background()
	rawCard := createTestEditableCard(t)
	exportTime := timestamp.Now[timestamp.Nano]()

	h1, _ := s.InsertCard(ctx, &models.Metadata{CardURL: "url1", DirectURL: "1", Source: source.ChubAI, UpdateTime: 1000}, rawCard, 1, 0)
	h2, _ := s.InsertCard(ctx, &models.Metadata{CardURL: "url2", DirectURL: "2", Source: source.Pygmalion, UpdateTime: 2000}, rawCard, 2, 0)
	h3, _ := s.InsertCard(ctx, &models.Metadata{CardURL: "url3", DirectURL: "3", Source: source.WyvernChat, UpdateTime: 3000}, rawCard, 3, 0)

	err := s.UpdateToLatestExport(ctx, h1.CardID, exportTime)
	require.NoError(t, err)

	err = s.UpdateToLatestExport(ctx, h2.CardID, exportTime)
	require.NoError(t, err)

	dbCard1, err := client.Card.Get(ctx, h1.CardID)
	require.NoError(t, err)
	assert.Equal(t, exportTime, dbCard1.ExportTime)
	assert.Equal(t, timestamp.Nano(1000), dbCard1.LastExportedVersion, "LastExportedVersion should be set to the card's UpdateTime")

	dbCard2, err := client.Card.Get(ctx, h2.CardID)
	require.NoError(t, err)
	assert.Equal(t, exportTime, dbCard2.ExportTime)
	assert.Equal(t, timestamp.Nano(2000), dbCard2.LastExportedVersion)

	dbCard3, err := client.Card.Get(ctx, h3.CardID)
	require.NoError(t, err)
	assert.Zero(t, dbCard3.ExportTime, "Card 3 should not have been exported")
	assert.Zero(t, dbCard3.LastExportedVersion, "Card 3 should not have a last exported version")
}

func TestService_UpdateStatus_Integration(t *testing.T) {
	s, client := setupIntegrationTest(t)
	ctx := context.Background()
	rawCard := createTestEditableCard(t)
	checkTime := timestamp.Now[timestamp.Nano]()

	header, err := s.InsertCard(context.Background(), &models.Metadata{CardURL: "url1", Source: source.ChubAI}, rawCard, 1, 0)
	require.NoError(t, err)

	err = s.UpdateStatus(context.Background(), header.CardID, checkTime, scheme.UpdateSuccess)
	require.NoError(t, err)

	dbCard, err := client.Card.Get(ctx, header.CardID)
	require.NoError(t, err)
	assert.Equal(t, scheme.UpdateSuccess, dbCard.LastUpdateStatus)
	assert.Equal(t, checkTime, dbCard.CheckTime)
}

func TestService_Favorite_Integration(t *testing.T) {
	s, client := setupIntegrationTest(t)
	ctx := context.Background()
	rawCard := createTestEditableCard(t)

	h1, err := s.InsertCard(ctx, &models.Metadata{CardURL: "url_fav_1", DirectURL: "fav_1", Source: source.ChubAI, CharacterID: "1", PlatformID: "1"}, rawCard, 1, 0)
	require.NoError(t, err)
	h2, err := s.InsertCard(ctx, &models.Metadata{CardURL: "url_fav_2", DirectURL: "fav_2", Source: source.Pygmalion, CharacterID: "2", PlatformID: "2"}, rawCard, 2, 1)
	require.NoError(t, err)
	h3, err := s.InsertCard(ctx, &models.Metadata{CardURL: "url_fav_3", DirectURL: "fav_3", Source: source.WyvernChat, CharacterID: "3", PlatformID: "3"}, rawCard, 3, 2)
	require.NoError(t, err)

	t.Run("ToggleFavorite", func(t *testing.T) {
		dbCard1, err := client.Card.Get(ctx, h1.CardID)
		require.NoError(t, err)
		assert.False(t, dbCard1.Favorite, "Card should start as not favorite")

		err = s.ToggleFavorite(ctx, h1.CardID)
		require.NoError(t, err)

		dbCard1, err = client.Card.Get(ctx, h1.CardID)
		require.NoError(t, err)
		assert.True(t, dbCard1.Favorite, "Card should be favorite after first toggle")

		err = s.ToggleFavorite(ctx, h1.CardID)
		require.NoError(t, err)

		dbCard1, err = client.Card.Get(ctx, h1.CardID)
		require.NoError(t, err)
		assert.False(t, dbCard1.Favorite, "Card should be not favorite after second toggle")
	})

	t.Run("SetFavorites", func(t *testing.T) {
		err := s.SetFavorites(ctx, []scheme.CardID{h2.CardID, h3.CardID}, true)
		require.NoError(t, err)

		dbCard2, err := client.Card.Get(ctx, h2.CardID)
		require.NoError(t, err)
		assert.True(t, dbCard2.Favorite)

		dbCard3, err := client.Card.Get(ctx, h3.CardID)
		require.NoError(t, err)
		assert.True(t, dbCard3.Favorite)

		dbCard1, err := client.Card.Get(ctx, h1.CardID)
		require.NoError(t, err)
		assert.False(t, dbCard1.Favorite)

		err = s.SetFavorites(ctx, []scheme.CardID{h2.CardID}, false)
		require.NoError(t, err)

		dbCard2, err = client.Card.Get(ctx, h2.CardID)
		require.NoError(t, err)
		assert.False(t, dbCard2.Favorite)

		dbCard3, err = client.Card.Get(ctx, h3.CardID)
		require.NoError(t, err)
		assert.True(t, dbCard3.Favorite)
	})
}
