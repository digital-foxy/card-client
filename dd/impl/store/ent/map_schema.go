package ent

import (
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/toolkit/slicesx"
)

func MapCardHeader(card *Card, header *scheme.CardHeader) *scheme.CardHeader {
	if card == nil {
		return nil
	}

	header.CardID = card.ID
	header.BatchOrder = card.BatchOrder
	header.Source = card.Source
	header.CardURL = card.CardURL
	header.DirectURL = card.DirectURL
	header.PlatformID = card.PlatformID
	header.CharacterID = card.CharacterID
	header.CardName = card.CardName
	header.CharacterName = card.CharacterName
	header.Creator = card.Creator
	header.Tagline = card.Tagline
	header.CreateTime = card.CreateTime
	header.UpdateTime = card.UpdateTime
	header.BookUpdateTime = card.BookUpdateTime
	header.Tags = slicesx.Map(card.Edges.Tags, MapTag)
	header.CheckTime = card.CheckTime
	header.LastUpdateStatus = card.LastUpdateStatus
	header.ExportTime = card.ExportTime
	header.LastExportedVersion = card.LastExportedVersion
	header.ImportTime = card.ImportTime
	header.Favorite = card.Favorite
	return header
}

func MapTag(tag *Tag) scheme.Tag {
	if tag == nil {
		return scheme.Tag{}
	}

	return scheme.Tag{
		ID:   tag.ID,
		Name: tag.Name,
	}
}
