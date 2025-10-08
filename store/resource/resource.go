package resource

import (
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/slicesx"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Box[T any] struct {
	Items     []T
	Timestamp timestamp.Nano
}

type Record struct {
	ID RID
	ImportData
	InfoData
	Creator
	SyncData
	ExportData
	AuxData
}

type ImportHeader struct {
	ID RID
	ImportData
}

type ImportData struct {
	ImportTime  timestamp.Nano
	ImportIndex int
}

type InfoHeader struct {
	ID RID
	InfoData
}

type InfoData struct {
	Source         source.ID
	NormalizedURL  string
	DirectURL      string
	PlatformID     string
	CharacterID    string
	Name           string
	Title          string
	Tagline        string
	CreateTime     timestamp.Nano
	UpdateTime     timestamp.Nano
	BookUpdateTime timestamp.Nano
	Tags           []Tag
}

type SyncHeader struct {
	ID RID
	SyncData
}

type SyncData struct {
	SyncTime   timestamp.Nano
	SyncStatus SyncStatus
}

type ExportHeader struct {
	ID RID
	ExportData
}

type ExportData struct {
	ExportTime      timestamp.Nano
	ExportedVersion timestamp.Nano
}

type AuxHeader struct {
	ID RID
	AuxData
}

type AuxData struct {
	Favorite bool
}

type Creator struct {
	ID         CID
	Nickname   string
	Username   string
	PlatformID string
	Source     source.ID
}

type Tag struct {
	ID   TID
	Name string
}

func TagNames(tags []Tag) []string {
	return slicesx.Map(tags, func(tag Tag) string {
		return tag.Name
	})
}

func TagIDs(tags []Tag) []TID {
	return slicesx.Map(tags, func(tag Tag) TID {
		return tag.ID
	})
}
