package catalog

import (
	"context"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Service interface {
	Label() string
	Count(ctx context.Context, filter resource.Filter) int
	FindPagedRIDs(ctx context.Context, filter resource.Filter, offset int, limit int) []resource.RID
	FindRecords(ctx context.Context, rids []resource.RID) resource.Box[resource.Record]
	FindExportHeaders(ctx context.Context, rids []resource.RID) resource.Box[resource.ExportHeader]
	FindURLs(ctx context.Context, normalizedURLs []string) []string

	// PNG operations
	GetPngPath(cardID scheme.CardID, version timestamp.Nano) string
	GetThumbnailPath(cardID string) string

	// Tag operations
	InsertStandardTags(ctx context.Context) error

	// Card operations
	InsertCard(ctx context.Context, metadata *models.Metadata, characterCard *png.CharacterCard, importTime timestamp.Nano, batchOrder int) (*scheme.CardHeader, error)
	UpdateCard(ctx context.Context, cardID scheme.CardID, metadata *models.Metadata, characterCard *png.CharacterCard, checkTime timestamp.Nano) (*scheme.CardHeader, error)
	UpdateStatus(ctx context.Context, cardID scheme.CardID, checkTime timestamp.Nano, status scheme.UpdateStatus) error
	UpdateToLatestExport(ctx context.Context, cardID scheme.CardID, exportTime timestamp.Nano) error

	// Favorite operations
	ToggleFavorite(ctx context.Context, cardID scheme.CardID) error
	SetFavorites(ctx context.Context, cardIDs []scheme.CardID, favorite bool) error

	Close() error
}
