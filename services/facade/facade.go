package facade

import (
	"github.com/r3dpixel/card-client/services/filter"
	"github.com/r3dpixel/card-client/services/operation"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/router"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/symbols"
	"github.com/r3dpixel/toolkit/timestamp"
)

type FileNameBuilder func(header scheme.MiscHeader) string

var DefaultFileNameBuilder = func(header scheme.MiscHeader) string {
	return filex.SanitizePath(string(header.Source)+symbols.Underscore+header.PlatformID) + png.Extension
}

type IntegrationReport struct {
	SourceID source.ID
	Status   router.IntegrationStatus
}

type Service interface {
	SetFileNameBuilder(builder FileNameBuilder)
	LoadVault(vault string) error
	UnloadVault() error
	ImportURLs(rawURLs string) (operation.ID, error)
	ExportLatestCards(cardIDs ...scheme.CardID) (operation.ID, error)
	UpdateCards(force bool, cardIDs ...scheme.CardID) (operation.ID, error)
	FindIDs(filter filter.SearchFilter) ([]scheme.CardID, error)
	FindPagedIDs(filter filter.SearchFilter, offset int, limit int) ([]scheme.CardID, error)
	FindCards(cardIDs ...scheme.CardID) ([]scheme.CardView, timestamp.Nano, error)
	ToggleFavorite(cardID scheme.CardID) error
	SetFavorites(cardIDs []scheme.CardID, favorite bool) error
	Count() (int, error)
	FlushUpdatePayloads() ([]scheme.UpdatePayload, timestamp.Nano, error)
	FlushExportPayloads() ([]scheme.ExportPayload, timestamp.Nano, error)
	HasUpdatePayloadRequests() bool
	HasExportPayloadRequests() bool
	GetSources() []source.ID
	GetSourceStatuses() map[source.ID]router.IntegrationStatus
}
