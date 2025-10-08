package record

import (
	"context"

	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
)

type Store interface {
	TxStore

	WithContext(ctx context.Context) Store
	WithTx(fn func(TxStore) error) error
	Close() error
}

type TxStore interface {
	Count(filter resource.Filter) int
	FindPagedRIDs(filter resource.Filter, offset int, limit int) []resource.RID
	FindRecords(rids []resource.RID) resource.Box[resource.Record]
	FindExportHeaders(rids []resource.RID) resource.Box[resource.ExportHeader]
	FindURLs(normalizedURLs []string) []string

	InsertRecord(metadata *models.Metadata, importData resource.ImportData) error
	UpdateRecord(rid resource.RID, metadata *models.Metadata, syncData resource.SyncData) error
	UpdateSyncData(rid resource.RID, syncData resource.SyncData) error
	UpdateExportData(rid resource.RID, exportData resource.ExportData) error
	UpdateFavoriteData(rids []resource.RID, favorite bool) error
	ToggleFavorite(rid resource.RID) error
}

func NewStore(opts any) Store {
	switch opts.(type) {

	}
	return nil
}
