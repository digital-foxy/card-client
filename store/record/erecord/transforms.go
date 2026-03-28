package erecord

import (
	"github.com/digital-foxy/card-client/store/record/erecord/ent"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/slicesx"
)

// mapRecord maps an ent RecordEntity to a resource Record
func mapRecord(entity *ent.RecordEntity, record *resource.Record) {
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
	record.GreetingsCount = entity.GreetingsCount
	record.IsFork = entity.IsFork

	entityTags := entity.Edges.Tags
	record.Tags = slicesx.Map(entityTags, transformTagEntity)

	if entity.Edges.Creator != nil {
		mapCreator(entity.Edges.Creator, &record.Creator)
	}

	record.SyncTime = entity.SyncTime
	record.SyncStatus = entity.SyncStatus

	record.ExportTime = entity.ExportTime
	record.ExportedVersion = entity.ExportedVersion

	record.Favorite = entity.Favorite
}

// mapCreator maps an ent CreatorEntity to a resource Creator
func mapCreator(entity *ent.CreatorEntity, creator *resource.Creator) {
	creator.ID = entity.ID
	creator.Nickname = entity.Nickname
	creator.Username = entity.Username
	creator.PlatformID = entity.PlatformID
	creator.Source = entity.Source
}

// transformCreator maps an ent CreatorEntity to a resource Creator
func transformCreator(entity *ent.CreatorEntity) resource.Creator {
	var creator resource.Creator
	mapCreator(entity, &creator)
	return creator
}

// mapTagEntity maps an ent TagEntity to a resource Tag
func mapTagEntity(entity *ent.TagEntity, tag *resource.Tag) {
	tag.ID = entity.ID
	tag.Name = entity.Name
}

// transformTagEntity maps an ent TagEntity to a resource Tag
func transformTagEntity(entity *ent.TagEntity) resource.Tag {
	return resource.Tag{
		ID:   entity.ID,
		Name: entity.Name,
	}
}
