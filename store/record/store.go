package record

import (
	"context"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Builder interface {
	Build(path string) (Store, error)
}

type Store interface {
	CtxStore

	WithContext(ctx context.Context) CtxStore
	Close() error
}

type CtxStore interface {
	TxStore
	WithTx(fn func(TxStore) error) error
}

type TxStore interface {
	Count(filter resource.Filter) (int, error)
	FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error)
	FindRecord(rid resource.RID) (*resource.Record, error)
	FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error)
	FindExportHeaders(rids ...resource.RID) (resource.Box[resource.ExportHeader], error)
	FindURLs(normalizedURLs ...string) ([]string, error)
	FindTagNames(tids ...resource.TID) ([]string, error)

	InsertRecord(metadata *models.Metadata, importData resource.ImportData) (resource.RID, error)
	UpdateRecord(rid resource.RID, metadata *models.Metadata, syncTime timestamp.Nano) error
	UpdateSyncData(rid resource.RID, syncData resource.SyncData) error
	UpdateExportData(rid resource.RID, exportData resource.ExportData) error

	UpdateFavoriteData(favorite bool, rids ...resource.RID) error
	ToggleFavorite(rid resource.RID) error
}
