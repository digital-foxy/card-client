package scheme

import (
	"github.com/r3dpixel/card-client/services/operation"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
)

type CardHeader struct {
	CardID     CardID `sql:"id"`
	BatchOrder int    `sql:"batch_order"`
	DataHeader
	UpdateHeader
	ExportHeader
	ImportTime timestamp.Nano `sql:"import_time"`
	Favorite   bool           `sql:"favorite"`
}

type CardView struct {
	CardHeader
	Thumbnail string
}

type UpdatePayload struct {
	CardID      CardID
	OperationID operation.ID
	DataHeader
	UpdateHeader
}

type ExportPayload struct {
	OperationID operation.ID
	CardID      CardID
	ExportHeader
}

type IdExportHeader struct {
	CardID CardID `sql:"id"`
	ExportHeader
}

type DataHeader struct {
	Source         source.ID      `sql:"source"`
	CardURL        string         `sql:"card_url"`
	DirectURL      string         `sql:"direct_url"`
	PlatformID     string         `sql:"platform_id"`
	CharacterID    string         `sql:"character_id"`
	CardName       string         `sql:"card_name"`
	CharacterName  string         `sql:"character_name"`
	Creator        string         `sql:"creator"`
	Tagline        string         `sql:"tagline"`
	CreateTime     timestamp.Nano `sql:"create_time"`
	UpdateTime     timestamp.Nano `sql:"update_time"`
	BookUpdateTime timestamp.Nano `sql:"book_update_time"`
	Tags           []Tag          `sql:"tags"`
}

type MiniHeader struct {
	CardID         CardID         `sql:"id"`
	CardURL        string         `sql:"card_url"`
	Creator        string         `sql:"creator"`
	UpdateTime     timestamp.Nano `sql:"update_time"`
	BookUpdateTime timestamp.Nano `sql:"book_update_time"`
}

type UpdateHeader struct {
	CheckTime        timestamp.Nano `sql:"check_time"`
	LastUpdateStatus UpdateStatus   `sql:"last_update_status"`
}

type ExportHeader struct {
	ExportTime          timestamp.Nano `sql:"export_time"`
	LastExportedVersion timestamp.Nano `sql:"last_exported_version"`
}

type MiscHeader struct {
	CardID        CardID         `sql:"id"`
	Source        source.ID      `sql:"source"`
	PlatformID    string         `sql:"platform_id"`
	CharacterID   string         `sql:"character_id"`
	CardName      string         `sql:"card_name"`
	CharacterName string         `sql:"character_name"`
	Creator       string         `sql:"creator"`
	UpdateTime    timestamp.Nano `sql:"update_time"`
}
