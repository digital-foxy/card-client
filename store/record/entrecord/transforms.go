package entrecord

import (
	"github.com/r3dpixel/card-client/store/record/entrecord/ent"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/slicesx"
)

func mapRecordEntity(entity *ent.RecordEntity, record *resource.Record) {
	record.ID = entity.ID

	record.ImportTime = entity.ImportTime
	record.ImportIndex = entity.ImportIndex

	record.InfoData.Source = entity.Source
	record.NormalizedURL = entity.NormalizedURL
	record.DirectURL = entity.DirectURL
	record.InfoData.PlatformID = entity.PlatformID
	record.CharacterID = entity.CharacterID
	record.Name = entity.Name
	record.Title = entity.Title
	record.Tagline = entity.Tagline
	record.CreateTime = entity.CreateTime
	record.UpdateTime = entity.UpdateTime
	record.BookUpdateTime = entity.BookUpdateTime

	entityTags := entity.Edges.Tags
	record.Tags = slicesx.Map(entityTags, transformTagEntity)

	if entity.Edges.Creator != nil {
		mapCreatorEntity(entity.Edges.Creator, &record.Creator)
	}

	record.SyncTime = entity.SyncTime
	record.SyncStatus = entity.SyncStatus

	record.ExportTime = entity.ExportTime
	record.ExportedVersion = entity.ExportedVersion

	record.Favorite = entity.Favorite
}

func mapCreatorEntity(entity *ent.CreatorEntity, creator *resource.Creator) {
	creator.ID = entity.ID
	creator.Nickname = entity.Nickname
	creator.Username = entity.Username
	creator.PlatformID = entity.PlatformID
	creator.Source = entity.Source
}

func mapTagEntity(entity *ent.TagEntity, tag *resource.Tag) {
	tag.ID = entity.ID
	tag.Name = entity.Name
}

func transformTagEntity(entity *ent.TagEntity) resource.Tag {
	return resource.Tag{
		ID:   entity.ID,
		Name: entity.Name,
	}
}
