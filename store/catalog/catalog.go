package catalog

import (
	"context"
	"errors"
	"image"

	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Catalog struct {
	record    record.Store
	blob      blob.Store
	ctxRecord record.CtxStore
	ctxBlob   blob.CtxStore
}

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

func (c *Catalog) FindExportHeaders(rids ...resource.RID) (resource.Box[resource.ExportHeader], error) {
	return c.ctxRecord.FindExportHeaders(rids...)
}

func (c *Catalog) FindURLs(normalizedURLs ...string) ([]string, error) {
	return c.ctxRecord.FindURLs(normalizedURLs...)
}

func (c *Catalog) InsertCard(metadata *models.Metadata, characterCard *png.CharacterCard, importData resource.ImportData) error {
	return c.ctxRecord.WithTx(func(store record.TxStore) error {
		rid, err := store.InsertRecord(metadata, importData)
		if err != nil {
			return err
		}

		rec, err := store.FindRecord(rid)
		if err != nil {
			return err
		}

		characterCard.Tags = resource.TagNames(rec.Tags)
		rawCard, err := characterCard.Encode()
		if err != nil {
			return err
		}

		return c.ctxBlob.Put(rec.ID, rec.UpdateTime, rawCard)
	})
}

func (c *Catalog) UpdateCard(rid resource.RID, metadata *models.Metadata, characterCard *png.CharacterCard, syncTime timestamp.Nano) error {
	return c.ctxRecord.WithTx(func(store record.TxStore) error {
		if err := store.UpdateRecord(rid, metadata, syncTime); err != nil {
			return err
		}

		rec, err := store.FindRecord(rid)
		if err != nil {
			return err
		}

		characterCard.Tags = resource.TagNames(rec.Tags)
		rawCard, err := characterCard.Encode()
		if err != nil {
			return err
		}

		return c.ctxBlob.Put(rec.ID, rec.UpdateTime, rawCard)
	})
}

func (c *Catalog) UpdateSyncData(rid resource.RID, syncData resource.SyncData) error {
	return c.record.UpdateSyncData(rid, syncData)
}

func (c *Catalog) UpdateExportData(rid resource.RID, exportData resource.ExportData) error {
	return c.record.UpdateExportData(rid, exportData)
}

func (c *Catalog) UpdateFavoriteData(favorite bool, rids ...resource.RID) error {
	return c.record.UpdateFavoriteData(favorite, rids...)
}

func (c *Catalog) ToggleFavorite(rid resource.RID) error {
	return c.record.ToggleFavorite(rid)
}

func (c *Catalog) GetRawCard(rid resource.RID, version timestamp.Nano) (*png.RawCard, error) {
	return c.ctxBlob.Get(rid, version)
}

func (c *Catalog) GetCardBytes(rid resource.RID, version timestamp.Nano) ([]byte, error) {
	return c.ctxBlob.GetBytes(rid, version)
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
