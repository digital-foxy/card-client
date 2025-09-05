package data

import (
	"context"

	"github.com/r3dpixel/card-client/store/resource"
)

type Store interface {
	Count(ctx context.Context, filter resource.Filter) int
	FindPagedRIDs(ctx context.Context, filter resource.Filter, offset int, limit int) []resource.RID
	FindRecords(ctx context.Context, rids []resource.RID) resource.Box[resource.Record]
	FindExportHeaders(ctx context.Context, rids []resource.RID) resource.Box[resource.ExportHeader]
	FindURLs(ctx context.Context, normalizedURLs []string) []string

	Insert(ctx context.Context, infoData *resource.InfoData, importData resource.ImportData) (*resource.Record, error)
	UpdateInfoSyncData(ctx context.Context, rid resource.RID, infoData *resource.InfoData, syncData resource.SyncData) (*resource.Record, error)
	UpdateSyncData(ctx context.Context, rid resource.RID, syncData resource.SyncData) error
	UpdateExportData(ctx context.Context, rid resource.RID, exportData resource.ExportData) error

	UpdateFavoriteData(ctx context.Context, rids []resource.RID, favorite bool) error
	ToggleFavorite(ctx context.Context, rid resource.RID) error
}
