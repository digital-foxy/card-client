package resource

import (
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Box[T any] struct {
	Items     []T
	Timestamp timestamp.Nano
}

type Record struct {
	CatalogID  CID
	ResourceID RID
	ImportData
	InfoData
	SyncData
	ExportData
	AuxData
}

type InfoData struct {
	Source         source.ID
	CardURL        string
	DirectURL      string
	PlatformID     string
	CharacterID    string
	CardName       string
	CharacterName  string
	Creator        string
	Tagline        string
	CreateTime     timestamp.Nano
	UpdateTime     timestamp.Nano
	BookUpdateTime timestamp.Nano
	Tags           []Tag
}

type SyncHeader struct {
	ResourceID RID
	SyncData
}

type SyncData struct {
	SyncTime       timestamp.Nano
	LastSyncStatus SyncStatus
}

type ExportHeader struct {
	ResourceID RID
	ExportData
}

type ExportData struct {
	ExportTime          timestamp.Nano
	LastExportedVersion timestamp.Nano
}

type ImportHeader struct {
	ResourceID RID
	ImportData
}

type ImportData struct {
	ImportTime  timestamp.Nano
	ImportIndex int
}

type AuxData struct {
	Favorite bool
}

type Tag struct {
	ID   TID
	Name string
}
