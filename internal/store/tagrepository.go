package store

import (
	"context"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/internal/ent/tag"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/models"
)

type tagRepository struct{}

func (r *tagRepository) upsertTags(client *ent.Client, ctx context.Context, tags []models.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	if err := client.Tag.MapCreateBulk(tags, func(c *ent.TagCreate, index int) {
		c.SetID(scheme.TagID(tags[index].Slug)).SetName(tags[index].Name)
	}).OnConflict().DoNothing().Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *tagRepository) findTagNames(client *ent.Client, ctx context.Context, tagIDs []scheme.TagID) []string {
	names, err := client.Tag.Query().
		Where(tag.IDIn(tagIDs...)).
		Order(ent.Asc(tag.FieldID)).
		Select(tag.FieldName).
		Strings(ctx)
	if err != nil {
		return nil
	}
	return names
}

func (r *tagRepository) findTags(client *ent.Client, ctx context.Context, tagIDs []scheme.TagID) []scheme.Tag {
	entTags, err := client.Tag.Query().
		Where(tag.IDIn(tagIDs...)).
		Order(ent.Asc(tag.FieldID)).
		All(ctx)
	if err != nil {
		return nil
	}
	tags := make([]scheme.Tag, len(entTags))
	for index, entTag := range entTags {
		tags[index] = ent.MapTag(entTag)
	}
	return tags
}

func (r *tagRepository) insertStandardTags(client *ent.Client, ctx context.Context) error {
	return r.upsertTags(client, ctx, models.TagsFromMap(models.StandardTags))
}
