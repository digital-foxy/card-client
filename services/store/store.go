package store

import (
	"context"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/filter"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-client/services/vault"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Service interface {
	VaultName() string
	Close() error
	Count(ctx context.Context) int
	FindPagedIDs(ctx context.Context, filter filter.SearchFilter, offset int, limit int) []scheme.CardID
	FindCards(ctx context.Context, cardIDs []scheme.CardID) ([]scheme.CardHeader, timestamp.Nano)
	FindIdExportHeaders(ctx context.Context, cardIDs []scheme.CardID) ([]scheme.IdExportHeader, timestamp.Nano)
	FindURLs(ctx context.Context, normalizedURLs []string) []string
	FindMiniHeaders(ctx context.Context, cardIDs []scheme.CardID) []scheme.MiniHeader
	FindMiniHeader(ctx context.Context, cardID scheme.CardID) (scheme.MiniHeader, error)
	FindMiscHeaders(ctx context.Context, cardIDs []scheme.CardID) []scheme.MiscHeader
	FindMiscHeader(ctx context.Context, cardID scheme.CardID) (scheme.MiscHeader, error)

	GetPngPath(cardID scheme.CardID, version timestamp.Nano) string
	GetThumbnailPath(cardID string) string
	InsertStandardTags(ctx context.Context) error
	InsertCard(ctx context.Context, metadata *models.Metadata, editableCard *png.CharacterCard, importTime timestamp.Nano, batchOrder int) (*scheme.CardHeader, error)
	UpdateCard(ctx context.Context, cardID scheme.CardID, metadata *models.Metadata, editableCard *png.CharacterCard, checkTime timestamp.Nano) (*scheme.CardHeader, error)
	UpdateStatus(ctx context.Context, cardID scheme.CardID, checkTime timestamp.Nano, status scheme.UpdateStatus) error
	UpdateToLatestExport(ctx context.Context, cardID scheme.CardID, exportTime timestamp.Nano) error
	ToggleFavorite(ctx context.Context, cardID scheme.CardID) error
	SetFavorites(ctx context.Context, cardIDs []scheme.CardID, favorite bool) error
}

type Provider = func(client *ent.Client, vault vault.Vault, opts opts.PngOptions) Service
