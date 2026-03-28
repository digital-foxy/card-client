package catalog

import (
	"context"
	"errors"
	"image"

	"github.com/digital-foxy/card-client/store/blob"
	"github.com/digital-foxy/card-client/store/record"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/structx"
	"github.com/digital-foxy/toolkit/timestamp"
)

 // Service provides unified access to record and blob stores
type Service interface {
	Count(filter resource.Filter) (int, error)
	FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error)
	FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error)
	FindRecord(rid resource.RID) (*resource.Record, error)
	FindCreatorByCID(cid resource.CID) (resource.Creator, error)
	FindCreatorByNickname(source source.ID, nickname string) (resource.Creator, error)
	FindURLs(normalizedURLs ...string) ([]string, error)
	TagNames() ([]string, error)

	CID(source source.ID, platformID string) resource.CID

	SaveCard(metadata *models.Metadata, characterCard *png.CharacterCard, time timestamp.Nano, importIndex ...int) (resource.RID, error)
	RestoreCard(rec *resource.Record, characterCard *png.CharacterCard) (resource.RID, error)
	UpdateSyncData(rid resource.RID, syncData resource.SyncData) error
	UpdateExportData(rid resource.RID, exportData resource.ExportData) error

	UpdateFavoriteData(favorite bool, rids ...resource.RID) error
	ToggleFavorite(rid resource.RID) error

	GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error)
	GetCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error)
	GetSheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error)
	GetSheetBytes(rid resource.RID, version timestamp.Nano) ([]byte, error)
	Thumbnail(rid resource.RID) (image.Image, error)
	ThumbnailBytes(rid resource.RID) ([]byte, error)
	CardVersions(rid resource.RID) []timestamp.Nano
	CardVersionExists(rid resource.RID, version timestamp.Nano) (bool, error)

	BasicIntegrity() ([]resource.RID, bool)
	FixRecordIntegrity(rec *resource.Record) resource.RecordIntegrity

	Delete(rids ...resource.RID) (int, error)
	CleanupCreators() (int, error)

	WithContext(ctx context.Context) Service
	Close() error
}

// Catalog implements Service using record and blob stores
type Catalog struct {
	record    record.Store
	blob      blob.Store
	ctxRecord record.CtxStore
	ctxBlob   blob.CtxStore
}

// New creates a new Catalog with the given stores
func New(recordStore record.Store, blobStore blob.Store) Service {
	return &Catalog{
		record:    recordStore,
		blob:      blobStore,
		ctxRecord: recordStore,
		ctxBlob:   blobStore,
	}
}

func (c *Catalog) Count(filter resource.Filter) (int, error) {
	return c.ctxRecord.Count(filter)
}

func (c *Catalog) FindPagedRIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error) {
	return c.ctxRecord.FindPagedRIDs(filter, offset, limit)
}

func (c *Catalog) FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error) {
	return c.ctxRecord.FindRecords(rids...)
}

func (c *Catalog) FindRecord(rid resource.RID) (*resource.Record, error) {
	return c.ctxRecord.FindRecord(rid)
}

func (c *Catalog) FindCreatorByCID(cid resource.CID) (resource.Creator, error) {
	return c.ctxRecord.FindCreatorByCID(cid)
}

func (c *Catalog) FindCreatorByNickname(source source.ID, nickname string) (resource.Creator, error) {
	return c.ctxRecord.FindCreatorByNickname(source, nickname)
}

func (c *Catalog) FindURLs(normalizedURLs ...string) ([]string, error) {
	return c.ctxRecord.FindURLs(normalizedURLs...)
}

func (c *Catalog) TagNames() ([]string, error) {
	return c.ctxRecord.TagNames()
}

func (c *Catalog) CID(source source.ID, platformID string) resource.CID {
	return c.ctxRecord.CID(source, platformID)
}

func (c *Catalog) SaveCard(metadata *models.Metadata, characterCard *png.CharacterCard, time timestamp.Nano, importIndex ...int) (resource.RID, error) {
	var rid resource.RID
	err := c.ctxRecord.WithTx(func(store record.TxStore) error {
		var err error
		rid, err = store.SaveRecord(metadata, characterCard, time, importIndex...)
		if err != nil {
			return err
		}

		rec, err := store.FindRecord(rid)
		if err != nil {
			return err
		}

		characterCard.Tags = resource.TagNames(rec.Tags)

		return c.ctxBlob.Put(rec.ID, rec.UpdateTime, characterCard)
	})
	return rid, err
}

func (c *Catalog) RestoreCard(rec *resource.Record, characterCard *png.CharacterCard) (resource.RID, error) {
	var rid resource.RID
	err := c.ctxRecord.WithTx(func(store record.TxStore) error {
		var err error
		rid, err = store.RestoreRecord(rec, characterCard)
		if err != nil {
			return err
		}

		characterCard.Tags = resource.TagNames(rec.Tags)

		return c.ctxBlob.Put(rid, rec.UpdateTime, characterCard)
	})
	return rid, err
}

func (c *Catalog) UpdateSyncData(rid resource.RID, syncData resource.SyncData) error {
	return c.ctxRecord.UpdateSyncData(rid, syncData)
}

func (c *Catalog) UpdateExportData(rid resource.RID, exportData resource.ExportData) error {
	return c.ctxRecord.UpdateExportData(rid, exportData)
}

func (c *Catalog) UpdateFavoriteData(favorite bool, rids ...resource.RID) error {
	return c.ctxRecord.UpdateFavoriteData(favorite, rids...)
}

func (c *Catalog) ToggleFavorite(rid resource.RID) error {
	return c.ctxRecord.ToggleFavorite(rid)
}

func (c *Catalog) GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	return c.ctxBlob.GetRawCard(rid, version)
}

func (c *Catalog) GetCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	return c.ctxBlob.GetRawCardBytes(rid, version)
}

func (c *Catalog) GetSheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error) {
	return c.ctxBlob.GetSheet(rid, version)
}

func (c *Catalog) GetSheetBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	return c.ctxBlob.GetSheetBytes(rid, version)
}

func (c *Catalog) Thumbnail(rid resource.RID) (image.Image, error) {
	return c.ctxBlob.Thumbnail(rid)
}

func (c *Catalog) ThumbnailBytes(rid resource.RID) ([]byte, error) {
	return c.ctxBlob.ThumbnailBytes(rid)
}

func (c *Catalog) CardVersions(rid resource.RID) []timestamp.Nano {
	return c.ctxBlob.Versions(rid)
}

func (c *Catalog) CardVersionExists(rid resource.RID, version timestamp.Nano) (bool, error) {
	return c.ctxBlob.VersionExists(rid, version)
}
func (c *Catalog) BasicIntegrity() ([]resource.RID, bool) {
	recordRIDs, err := c.ctxRecord.FindPagedRIDs(resource.Filter{}, 0, -1)
	if err != nil {
		return nil, false
	}
	blobRIDs, err := c.ctxBlob.RIDs()
	if err != nil {
		return nil, false
	}
	if len(recordRIDs) != len(blobRIDs) {
		return nil, false
	}
	ridMap := make(map[resource.RID]struct{})
	for _, rid := range recordRIDs {
		ridMap[rid] = structx.Empty
	}
	if len(ridMap) != len(blobRIDs) || len(ridMap) != len(recordRIDs) {
		return nil, false
	}

	return recordRIDs, true
}

func (c *Catalog) FixRecordIntegrity(rec *resource.Record) resource.RecordIntegrity {
	characterCard, err := c.ctxBlob.GetCharacterCard(rec.ID, rec.UpdateTime)
	if err != nil {
		return resource.BROKEN
	}

	recordIntegrity := rec.FixIntegrity(characterCard.Sheet)
	if recordIntegrity == resource.BROKEN {
		return resource.BROKEN
	}

	if !c.isThumbnailOk(rec.ID) {
		recordIntegrity = resource.FIXED
	}
	if recordIntegrity == resource.OK {
		return resource.OK
	}

	if _, err := c.SaveCard(rec.ToMetadata(), characterCard, rec.UpdateTime); err != nil {
		return resource.BROKEN
	}

	characterCard, cardErr := c.ctxBlob.GetCharacterCard(rec.ID, rec.UpdateTime)
	if cardErr != nil || !c.isThumbnailOk(rec.ID) || !characterCard.Sheet.Integrity() {
		return resource.BROKEN
	}

	return resource.FIXED
}

func (c *Catalog) Delete(rids ...resource.RID) (int, error) {
	var deleted int

	txErr := c.ctxRecord.WithTx(func(store record.TxStore) error {
		var err error
		deleted, err = store.Delete(rids...)
		if err != nil {
			return err
		}

		return c.ctxBlob.WithTx(func(store blob.TxStore) error {
			for _, rid := range rids {
				if err := store.Delete(rid); err != nil {
					return err
				}
			}
			return nil
		})
	})
	if txErr != nil {
		return 0, txErr
	}

	return deleted, nil
}

func (c *Catalog) CleanupCreators() (int, error) {
	return c.ctxRecord.CleanupCreators()
}

func (c *Catalog) isThumbnailOk(rid resource.RID) bool {
	thumbnail, err := c.ctxBlob.Thumbnail(rid)
	if err != nil {
		return false
	}
	bounds := thumbnail.Bounds()
	return bounds.Dx() > 0 && bounds.Dy() > 0
}

func (c *Catalog) WithContext(ctx context.Context) Service {
	ctxCatalog := *c
	ctxCatalog.ctxRecord = ctxCatalog.record.WithContext(ctx)
	ctxCatalog.ctxBlob = ctxCatalog.blob.WithContext(ctx)
	return &ctxCatalog
}

func (c *Catalog) Close() error {
	return errors.Join(
		c.record.Close(),
		c.blob.Close(),
	)
}
