package resource

import (
	"github.com/r3dpixel/card-fetcher/models"
)

func FromMetadata(metadata *models.Metadata) *InfoData {
	tags := make([]Tag, len(metadata.Tags))
	for i, tag := range metadata.Tags {
		tags[i] = Tag{
			ID:   TID(tag.Slug),
			Name: tag.Name,
		}
	}

	return &InfoData{
		Source:         metadata.Source,
		CardURL:        metadata.CardURL,
		DirectURL:      metadata.DirectURL,
		PlatformID:     metadata.PlatformID,
		CharacterID:    metadata.CharacterID,
		CardName:       metadata.CardName,
		CharacterName:  metadata.CharacterName,
		Creator:        metadata.Creator,
		Tagline:        metadata.Tagline,
		CreateTime:     metadata.CreateTime,
		UpdateTime:     metadata.UpdateTime,
		BookUpdateTime: metadata.BookUpdateTime,
		Tags:           tags,
	}
}

func TagNames(tags []Tag) []string {
	names := make([]string, len(tags))
	for index, tag := range tags {
		names[index] = tag.Name
	}
	return names
}
