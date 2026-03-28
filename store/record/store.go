package record

import (
	"context"

	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/timestamp"
)

// Builder creates record stores from a path
type Builder interface {
	Build(path string) (Store, error)
}

// Store is the main record storage interface
type Store interface {
	CtxStore

	WithContext(ctx context.Context) CtxStore
	Close() error
}

// CtxStore is a context-aware record store
type CtxStore interface {
	TxStore
	WithTx(fn func(TxStore) error) error
}

// TxStore provides transactional record operations
type TxStore interface {
	Count(filter resource.Filter) (int, error)
	FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error)
	FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error)
	FindRecord(rid resource.RID) (*resource.Record, error)
	FindCreatorByCID(cid resource.CID) (resource.Creator, error)
	FindCreatorByNickname(source source.ID, nickname string) (resource.Creator, error)
	FindURLs(normalizedURLs ...string) ([]string, error)
	FindTagNames(tids ...resource.TID) ([]string, error)
	TagNames() ([]string, error)

	CID(source source.ID, platformID string) resource.CID
	UpsertTags(tags []models.Tag) error
	UpsertCreator(source source.ID, creatorInfo models.CreatorInfo) (resource.CID, error)

	SaveRecord(metadata *models.Metadata, characterCard *png.CharacterCard, time timestamp.Nano, importIndex ...int) (resource.RID, error)
	RestoreRecord(rec *resource.Record, characterCard *png.CharacterCard) (resource.RID, error)
	UpdateSyncData(rid resource.RID, syncData resource.SyncData) error
	UpdateExportData(rid resource.RID, exportData resource.ExportData) error

	UpdateFavoriteData(favorite bool, rids ...resource.RID) error
	ToggleFavorite(rid resource.RID) error

	Delete(rids ...resource.RID) (int, error)

	CleanupCreators() (int, error)
}
