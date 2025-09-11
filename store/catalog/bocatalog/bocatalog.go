package bocatalog

import (
	"context"

	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
)

type BoCatalog struct {
	blobStore   blob.Store
	recordStore record.Store
}

func New() *BoCatalog {
	return &BoCatalog{}
}

func (b *BoCatalog) Label() string {

}

func (b *BoCatalog) Count(ctx context.Context, filter resource.Filter) int {
	return b.recordStore.Count(ctx, filter)
}

func (b *BoCatalog) FindPagedRIDs(ctx context.Context, filter resource.Filter, offset int, limit int) []resource.RID {
	return b.recordStore.FindPagedRIDs(ctx, filter, offset, limit)
}

func (b *BoCatalog) FindRecords(ctx context.Context, rids []resource.RID) resource.Box[resource.Record] {
	return b.recordStore.FindRecords(ctx, rids)
}

func (b *BoCatalog) FindExportHeaders(ctx context.Context, rids []resource.RID) resource.Box[resource.ExportHeader] {
	return b.recordStore.FindExportHeaders(ctx, rids)
}

func (b *BoCatalog) FindURLs(ctx context.Context, normalizedURLs []string) []string {
	return b.recordStore.FindURLs(ctx, normalizedURLs)
}

func (b *BoCatalog) InsertCard(ctx context.Context, infoData *resource.InfoData, characterCard *png.CharacterCard, importData resource.ImportData) (*resource.Record, error) {
	rec, err := b.recordStore.Insert(ctx, infoData, importData)
	if err != nil {
		return nil, err
	}

	characterCard.Data.Tags = resource.TagNames(rec.Tags)
	rawCard, err := characterCard.Encode()
	if err != nil {
		return nil, err
	}

	err = b.blobStore.Put(rec.ResourceID, rec.UpdateTime, rawCard)
	if err != nil {
		return nil, err
	}

	return rec, nil
}

func (b *BoCatalog) UpdateCard(ctx context.Context, rid resource.RID, infoData *resource.InfoData, characterCard *png.CharacterCard, syncTime resource.SyncData) (*resource.Record, error) {
	rec, err := b.recordStore.UpdateInfoSyncData(ctx, rid, infoData, syncTime)
	if err != nil {
		return nil, err
	}

	characterCard.Data.Tags = resource.TagNames(rec.Tags)
	rawCard, err := characterCard.Encode()
	if err != nil {
		return nil, err
	}

	err = b.blobStore.Put(rec.ResourceID, rec.UpdateTime, rawCard)
	if err != nil {
		return nil, err
	}

	return rec, nil
}

func (b *BoCatalog) UpdateSyncData(ctx context.Context, rid resource.RID, syncData resource.SyncData) error {
	return b.recordStore.UpdateSyncData(ctx, rid, syncData)
}

func (b *BoCatalog) UpdateExportData(ctx context.Context, rid resource.RID, exportData resource.ExportData) error {
	return b.recordStore.UpdateExportData(ctx, rid, exportData)
}

func (b *BoCatalog) UpdateFavoriteData(ctx context.Context, rids []resource.RID, favorite bool) error {
	return b.recordStore.UpdateFavoriteData(ctx, rids, favorite)
}

func (b *BoCatalog) ToggleFavorite(ctx context.Context, rid resource.RID) error {
	return b.recordStore.ToggleFavorite(ctx, rid)
}

func (b *BoCatalog) Close() error {
	return nil
}
