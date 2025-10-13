package facade

import (
	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/symbols"
)

type FileNameBuilder func(rec *resource.Record) string

var DefaultFileNameBuilder = func(rec *resource.Record) string {
	return filex.SanitizePath(string(rec.InfoData.Source)+symbols.Underscore+rec.InfoData.PlatformID) + png.Extension
}

type OperationPayload struct {
	OperationID operation.ID
	Record      resource.Record
}

type IntegrationReport struct {
	SourceID source.ID
	Status   router.IntegrationStatus
}

type Service interface {
	SetFileNameBuilder(builder FileNameBuilder)
	LoadVault(vault library.VaultName) error
	UnloadVault() error

	CountRecords(filter resource.Filter) (int, error)
	FindRIDs(filter resource.Filter) ([]resource.RID, error)
	FindPagedIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error)
	FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error)

	ImportURLs(rawURLs string) (operation.ID, error)
	ExportLatestCards(rids ...resource.RID) (operation.ID, error)
	UpdateCards(force bool, rids ...resource.RID) (operation.ID, error)

	ToggleFavorite(rid resource.RID) error
	UpdateFavorites(favorite bool, rids ...resource.RID) error

	FlushUpdateCache() (resource.Box[OperationPayload], error)
	FlushExportCache() (resource.Box[OperationPayload], error)
	HasUpdatePayloadRequests() bool
	HasExportPayloadRequests() bool

	GetSources() []source.ID
	GetSourceStatuses() map[source.ID]router.IntegrationStatus
}
