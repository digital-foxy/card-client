package store

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/internal/ent/card"
	"github.com/r3dpixel/card-client/internal/ent/tag"
	"github.com/r3dpixel/card-client/services/filter"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/toolkit/slicesx"
	"github.com/r3dpixel/toolkit/timestamp"
)

type cardRepository struct {
	EntFilterBuilder
}

func newCardRepository() cardRepository {
	return cardRepository{
		EntFilterBuilder: NewFilterBuilder(),
	}
}

func (r *cardRepository) count(client *ent.Client, ctx context.Context) int {
	count, _ := client.Card.Query().Count(ctx)
	return count
}

func (r *cardRepository) findPagedCardIDs(client *ent.Client, ctx context.Context, filter filter.SearchFilter, offset int, limit int) []scheme.CardID {
	query, err := r.EntFilterBuilder.ApplyFilter(client.Card.Query(), filter)
	if err != nil {
		return []scheme.CardID{}
	}
	cardIDs, _ := query.
		Offset(offset).
		Limit(limit).
		IDs(ctx)

	return cardIDs
}

func (r *cardRepository) findCards(client *ent.Client, ctx context.Context, cardIDs []scheme.CardID) ([]scheme.CardHeader, timestamp.Nano) {
	readAt := timestamp.Now[timestamp.Nano]()

	cards, _ := client.Card.
		Query().
		WithTags(func(q *ent.TagQuery) {
			q.Order(ent.Asc(tag.FieldID))
		}).
		Where(card.IDIn(cardIDs...)).
		All(ctx)

	headers := make([]scheme.CardHeader, len(cards))
	for index, card := range cards {
		ent.MapCardHeader(card, &headers[index])
	}

	return headers, readAt
}

func (r *cardRepository) findExportPayloads(client *ent.Client, ctx context.Context, cardIDs []scheme.CardID) ([]scheme.IdExportHeader, timestamp.Nano) {
	readAt := timestamp.Now[timestamp.Nano]()
	var payloads []scheme.IdExportHeader

	_ = client.Card.
		Query().
		Where(card.IDIn(cardIDs...)).
		Select(card.FieldID, card.FieldExportTime, card.FieldLastExportedVersion).
		Scan(ctx, &payloads)

	return payloads, readAt
}

func (r *cardRepository) findURLs(client *ent.Client, ctx context.Context, normalizedURLs []string) []string {
	existingURLs, _ := client.Card.
		Query().
		Where(card.CardURLIn(normalizedURLs...)).
		Select(card.FieldCardURL).
		Strings(ctx)
	return existingURLs
}

func (r *cardRepository) findMiniHeaders(client *ent.Client, ctx context.Context, cardIDs []scheme.CardID) []scheme.MiniHeader {
	var headers []scheme.MiniHeader
	_ = client.Card.
		Query().
		Where(card.IDIn(cardIDs...)).
		Select(card.FieldID, card.FieldCardURL, card.FieldCreator, card.FieldUpdateTime, card.FieldBookUpdateTime).
		Scan(ctx, &headers)

	return headers
}

func (r *cardRepository) findMiniHeader(client *ent.Client, ctx context.Context, cardID scheme.CardID) (scheme.MiniHeader, error) {
	var headers []scheme.MiniHeader

	err := client.Card.
		Query().
		Where(card.ID(cardID)).
		Select(card.FieldID, card.FieldCardURL, card.FieldCreator, card.FieldUpdateTime, card.FieldBookUpdateTime).
		Scan(ctx, &headers)

	if err != nil {
		return scheme.MiniHeader{}, fmt.Errorf("failed to find mini header for card ResourceID %v: %w", cardID, err)
	}

	if len(headers) == 0 {
		return scheme.MiniHeader{}, fmt.Errorf("card not found")
	}

	return headers[0], nil
}

func (r *cardRepository) findMiscHeaders(client *ent.Client, ctx context.Context, cardIDs []scheme.CardID) []scheme.MiscHeader {
	var headers []scheme.MiscHeader
	_ = client.Card.
		Query().
		Where(card.IDIn(cardIDs...)).
		Select(card.FieldID, card.FieldSource, card.FieldPlatformID, card.FieldCharacterID, card.FieldCardName, card.FieldCharacterName, card.FieldCreator, card.FieldUpdateTime).
		Scan(ctx, &headers)

	return headers
}

func (r *cardRepository) findMiscHeader(client *ent.Client, ctx context.Context, cardID scheme.CardID) (scheme.MiscHeader, error) {
	var headers []scheme.MiscHeader

	err := client.Card.
		Query().
		Where(card.ID(cardID)).
		Select(card.FieldID, card.FieldSource, card.FieldPlatformID, card.FieldCharacterID, card.FieldCardName, card.FieldCharacterName, card.FieldCreator, card.FieldUpdateTime).
		Scan(ctx, &headers)

	if err != nil {
		return scheme.MiscHeader{}, fmt.Errorf("failed to find misc header for card ResourceID %v: %w", cardID, err)
	}

	if len(headers) == 0 {
		return scheme.MiscHeader{}, fmt.Errorf("card not found")
	}

	return headers[0], nil
}

func (r *cardRepository) insertCard(
	client *ent.Client,
	ctx context.Context,
	metadata *models.Metadata,
	tagIDs []scheme.TagID,
	importTime timestamp.Nano,
	batchOrder int,
) (*scheme.CardHeader, error) {
	card, err := client.Card.Create().
		SetBatchOrder(batchOrder).
		SetSource(metadata.Source).
		SetCardURL(metadata.CardURL).
		SetDirectURL(metadata.DirectURL).
		SetPlatformID(metadata.PlatformID).
		SetCharacterID(metadata.CharacterID).
		SetCardName(metadata.CardName).
		SetCharacterName(metadata.CharacterName).
		SetCreator(metadata.Creator).
		SetTagline(metadata.Tagline).
		SetCreateTime(metadata.CreateTime).
		SetUpdateTime(metadata.UpdateTime).
		SetBookUpdateTime(metadata.BookUpdateTime).
		SetImportTime(importTime).
		SetCheckTime(importTime).
		SetLastUpdateStatus(scheme.UpdateSuccess).
		AddTagIDs(tagIDs...).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	var header scheme.CardHeader
	ent.MapCardHeader(card, &header)
	header.Tags = slicesx.Map(metadata.Tags, func(t models.Tag) scheme.Tag {
		return scheme.Tag{
			ID:   scheme.TagID(t.Slug),
			Name: t.Name,
		}
	})
	return &header, nil
}

func (r *cardRepository) updateCard(
	client *ent.Client,
	ctx context.Context,
	cardID scheme.CardID,
	metadata *models.Metadata,
	tagIDs []scheme.TagID,
	updateHeader scheme.UpdateHeader,
) (*scheme.CardHeader, error) {
	entCard, err := client.Card.UpdateOneID(cardID).
		SetSource(metadata.Source).
		SetCardURL(metadata.CardURL).
		SetDirectURL(metadata.DirectURL).
		SetPlatformID(metadata.PlatformID).
		SetCharacterID(metadata.CharacterID).
		SetCardName(metadata.CardName).
		SetCharacterName(metadata.CharacterName).
		SetCreator(metadata.Creator).
		SetTagline(metadata.Tagline).
		SetCreateTime(metadata.CreateTime).
		SetUpdateTime(metadata.UpdateTime).
		SetBookUpdateTime(metadata.BookUpdateTime).
		SetCheckTime(updateHeader.CheckTime).
		SetLastUpdateStatus(scheme.UpdateSuccess).
		ClearTags().
		AddTagIDs(tagIDs...).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	var header scheme.CardHeader
	ent.MapCardHeader(entCard, &header)
	header.Tags = slicesx.Map(metadata.Tags, func(t models.Tag) scheme.Tag {
		return scheme.Tag{
			ID:   scheme.TagID(t.Slug),
			Name: t.Name,
		}
	})
	return &header, nil
}

func (r *cardRepository) updateCardStatus(
	client *ent.Client,
	ctx context.Context,
	cardID scheme.CardID,
	checkTime timestamp.Nano,
	status scheme.UpdateStatus,
) error {
	_, err := client.Card.
		UpdateOneID(cardID).
		SetCheckTime(checkTime).
		SetLastUpdateStatus(status).
		Save(ctx)

	return err
}

func (r *cardRepository) updateCardExportData(
	client *ent.Client,
	ctx context.Context,
	cardID scheme.CardID,
	exportData scheme.ExportHeader,
) error {
	_, err := client.Card.
		UpdateOneID(cardID).
		SetExportTime(exportData.ExportTime).
		SetLastExportedVersion(exportData.LastExportedVersion).
		Save(ctx)

	return err
}

func (r *cardRepository) updateToLatestExport(
	client *ent.Client,
	ctx context.Context,
	cardID scheme.CardID,
	exportTime timestamp.Nano,
) error {
	_, err := client.Card.
		Update().
		Where(card.ID(cardID)).
		SetExportTime(exportTime).
		Modify(func(u *sql.UpdateBuilder) {
			u.Set(card.FieldLastExportedVersion, sql.Expr(card.FieldUpdateTime))
		}).
		Save(ctx)

	return err
}

func (r *cardRepository) toggleFavorite(client *ent.Client, ctx context.Context, cardID scheme.CardID) error {
	return client.Card.
		UpdateOneID(cardID).
		Modify(func(u *sql.UpdateBuilder) {
			u.Set(card.FieldFavorite, sql.Expr("NOT "+card.FieldFavorite))
		}).
		Exec(ctx)
}

func (r *cardRepository) setFavorites(client *ent.Client, ctx context.Context, cardIDs []scheme.CardID, favorite bool) error {
	_, err := client.Card.
		Update().
		Where(card.IDIn(cardIDs...)).
		SetFavorite(favorite).
		Save(ctx)
	return err
}
