package card

import (
	"context"

	"github.com/r3dpixel/card-client/store/blob"
	"github.com/r3dpixel/card-client/store/record"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
)

type Catalog struct {
	blobStore   blob.Store
	recordStore record.Store
}

func New() *Catalog {
	return &Catalog{}
}

func (b *Catalog) Label() string {
	return "catalog"
}

func (b *Catalog) Count(ctx context.Context, filter resource.Filter) int {
	return b.recordStore.Count(ctx, filter)
}

func (b *Catalog) FindPagedRIDs(ctx context.Context, filter resource.Filter, offset int, limit int) []resource.RID {
	return b.recordStore.FindPagedRIDs(ctx, filter, offset, limit)
}

func (b *Catalog) FindRecords(ctx context.Context, rids []resource.RID) resource.Box[resource.Record] {
	return b.recordStore.FindRecords(ctx, rids)
}

func (b *Catalog) FindExportHeaders(ctx context.Context, rids []resource.RID) resource.Box[resource.ExportHeader] {
	return b.recordStore.FindExportHeaders(ctx, rids)
}

func (b *Catalog) FindURLs(ctx context.Context, normalizedURLs []string) []string {
	return b.recordStore.FindURLs(ctx, normalizedURLs)
}

func (b *Catalog) InsertCard(ctx context.Context, infoData *resource.InfoData, characterCard *png.CharacterCard, importData resource.ImportData) (*resource.Record, error) {
	rec, err := b.recordStore.Insert(ctx, infoData, importData)
	if err != nil {
		return nil, err
	}

	characterCard.Data.Tags = resource.TagNames(rec.Tags)
	rawCard, err := characterCard.Encode()
	if err != nil {
		return nil, err
	}

	err = b.blobStore.Put(rec.ID, rec.UpdateTime, rawCard)
	if err != nil {
		return nil, err
	}

	return rec, nil
}

func (b *Catalog) UpdateCard(ctx context.Context, rid resource.RID, infoData *resource.InfoData, characterCard *png.CharacterCard, syncTime resource.SyncData) (*resource.Record, error) {
	rec, err := b.recordStore.UpdateInfoSyncData(ctx, rid, infoData, syncTime)
	if err != nil {
		return nil, err
	}

	characterCard.Data.Tags = resource.TagNames(rec.Tags)
	rawCard, err := characterCard.Encode()
	if err != nil {
		return nil, err
	}

	err = b.blobStore.Put(rec.ID, rec.UpdateTime, rawCard)
	if err != nil {
		return nil, err
	}

	return rec, nil
}

func (b *Catalog) UpdateSyncData(ctx context.Context, rid resource.RID, syncData resource.SyncData) error {
	return b.recordStore.UpdateSyncData(ctx, rid, syncData)
}

func (b *Catalog) UpdateExportData(ctx context.Context, rid resource.RID, exportData resource.ExportData) error {
	return b.recordStore.UpdateExportData(ctx, rid, exportData)
}

func (b *Catalog) UpdateFavoriteData(ctx context.Context, rids []resource.RID, favorite bool) error {
	return b.recordStore.UpdateFavoriteData(ctx, rids, favorite)
}

func (b *Catalog) ToggleFavorite(ctx context.Context, rid resource.RID) error {
	return b.recordStore.ToggleFavorite(ctx, rid)
}

func (b *Catalog) Close() error {
	return nil
}
