package catalog

import (
	"context"
	"image"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Service interface {
	Count(filter resource.Filter) (int, error)
	FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error)
	FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error)
	FindRecord(rid resource.RID) (*resource.Record, error)
	FindExportHeaders(rids ...resource.RID) (resource.Box[resource.ExportHeader], error)
	FindURLs(normalizedURLs ...string) ([]string, error)

	InsertCard(metadata *models.Metadata, characterCard *png.CharacterCard, importData resource.ImportData) error
	UpdateCard(rid resource.RID, metadata *models.Metadata, characterCard *png.CharacterCard, syncTime timestamp.Nano) error
	UpdateSyncData(rid resource.RID, syncData resource.SyncData) error
	UpdateExportData(rid resource.RID, exportData resource.ExportData) error

	UpdateFavoriteData(favorite bool, rids ...resource.RID) error
	ToggleFavorite(rid resource.RID) error

	GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error)
	GetCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error)
	Thumbnail(rid resource.RID) (image.Image, error)
	ThumbnailBytes(rid resource.RID) ([]byte, error)
	CardVersions(rid resource.RID) []timestamp.Nano
	CardVersionExists(rid resource.RID, version timestamp.Nano) (bool, error)

	WithContext(ctx context.Context) Service
	Close() error
}
