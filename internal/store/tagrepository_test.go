package store

import (
	"context"
	"testing"

	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagRepository_UpsertTags(t *testing.T) {
	t.Run("inserts new tags successfully", func(t *testing.T) {
		client := newTestClient(t)
		ctx := context.Background()
		repo := &tagRepository{}

		tagsToInsert := []models.Tag{
			{Slug: "tag-a", Name: "Tag A"},
			{Slug: "tag-b", Name: "Tag B"},
		}

		err := repo.upsertTags(client, ctx, tagsToInsert)
		require.NoError(t, err)

		allTags, err := client.Tag.Query().All(ctx)
		require.NoError(t, err)
		assert.Len(t, allTags, 2)
	})

	t.Run("ignores conflicting tags and inserts new ones", func(t *testing.T) {
		client := newTestClient(t)
		ctx := context.Background()
		repo := &tagRepository{}

		initialTags := []models.Tag{
			{Slug: "tag-a", Name: "Tag A Original"},
		}
		require.NoError(t, repo.upsertTags(client, ctx, initialTags))

		tagsToUpsert := []models.Tag{
			{Slug: "tag-a", Name: "Tag A Updated"},
			{Slug: "tag-b", Name: "Tag B New"},
		}
		err := repo.upsertTags(client, ctx, tagsToUpsert)
		require.NoError(t, err)

		tagA, err := client.Tag.Get(ctx, scheme.TagID("tag-a"))
		require.NoError(t, err)
		assert.Equal(t, "Tag A Original", tagA.Name)

		tagB, err := client.Tag.Get(ctx, scheme.TagID("tag-b"))
		require.NoError(t, err)
		assert.Equal(t, "Tag B New", tagB.Name)

		count, err := client.Tag.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("handles an empty slice with no error", func(t *testing.T) {
		client := newTestClient(t)
		ctx := context.Background()
		repo := &tagRepository{}

		err := repo.upsertTags(client, ctx, []models.Tag{})
		require.NoError(t, err)

		count, err := client.Tag.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestTagRepository_InsertStandardTags(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()
	repo := &tagRepository{}

	err := repo.insertStandardTags(client, ctx)
	require.NoError(t, err)

	count, err := client.Tag.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(models.StandardTags), count)

	for slug, name := range models.StandardTags {
		oneTag, err := client.Tag.Get(ctx, scheme.TagID(slug))
		require.NoError(t, err)
		assert.Equal(t, name, oneTag.Name)
	}
}
