package store

import (
	"context"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/filter"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/slicesx"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
)

type Service struct {
	client         *ent.Client
	vault          string
	cardRepository cardRepository
	tagRepository  tagRepository
	pngRepository  pngRepository
}

func NewService(client *ent.Client, vault string, cardsDir string, opts opts.PngOptions) *Service {
	return &Service{
		client:         client,
		vault:          vault,
		cardRepository: newCardRepository(),
		tagRepository:  tagRepository{},
		pngRepository:  newPngRepository(cardsDir, opts),
	}
}

func (s *Service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func (s *Service) VaultName() string {
	return s.vault
}

func (s *Service) Count(ctx context.Context) int {
	return s.cardRepository.count(s.client, ctx)
}

func (s *Service) FindPagedIDs(ctx context.Context, filter filter.SearchFilter, offset int, limit int) []scheme.CardID {
	return s.cardRepository.findPagedCardIDs(s.client, ctx, filter, offset, limit)
}

func (s *Service) FindCards(ctx context.Context, cardIDs []scheme.CardID) ([]scheme.CardHeader, timestamp.Nano) {
	return s.cardRepository.findCards(s.client, ctx, cardIDs)
}

func (s *Service) FindIdExportHeaders(ctx context.Context, cardIDs []scheme.CardID) ([]scheme.IdExportHeader, timestamp.Nano) {
	return s.cardRepository.findExportPayloads(s.client, ctx, cardIDs)
}

func (s *Service) FindURLs(ctx context.Context, normalizedURLs []string) []string {
	return s.cardRepository.findURLs(s.client, ctx, normalizedURLs)
}

func (s *Service) FindMiniHeaders(ctx context.Context, cardIDs []scheme.CardID) []scheme.MiniHeader {
	return s.cardRepository.findMiniHeaders(s.client, ctx, cardIDs)
}

func (s *Service) FindMiniHeader(ctx context.Context, cardID scheme.CardID) (scheme.MiniHeader, error) {
	return s.cardRepository.findMiniHeader(s.client, ctx, cardID)
}

func (s *Service) FindMiscHeaders(ctx context.Context, cardIDs []scheme.CardID) []scheme.MiscHeader {
	return s.cardRepository.findMiscHeaders(s.client, ctx, cardIDs)
}

func (s *Service) FindMiscHeader(ctx context.Context, cardID scheme.CardID) (scheme.MiscHeader, error) {
	return s.cardRepository.findMiscHeader(s.client, ctx, cardID)
}

func (s *Service) GetPngPath(cardID scheme.CardID, version timestamp.Nano) string {
	return s.pngRepository.getCardPngPath(cardID, version)
}

func (s *Service) GetThumbnailPath(cardID string) string {
	return s.pngRepository.getThumbnailPath(cardID)
}

func (s *Service) InsertStandardTags(ctx context.Context) error {
	return s.tagRepository.insertStandardTags(s.client, ctx)
}

func (s *Service) InsertCard(ctx context.Context, metadata *models.Metadata, characterCard *png.CharacterCard, importTime timestamp.Nano, batchOrder int) (*scheme.CardHeader, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, trace.Err().
			Field(trace.ACTIVITY, "insert-card").
			Field(trace.URL, metadata.CardURL).
			Msg("Failed to initialize transaction)")
	}

	txClient := tx.Client()
	defer tx.Rollback()

	if err = s.tagRepository.upsertTags(txClient, ctx, metadata.Tags); err != nil {
		return nil, trace.Err().Wrap(err).
			Field(trace.ACTIVITY, "insert-card").
			Field(trace.URL, metadata.CardURL).
			Msg("Failed to insert tags")
	}
	tagIDs := slicesx.Map(metadata.Tags, func(tag models.Tag) scheme.TagID {
		return scheme.TagID(tag.Slug)
	})

	cardHeader, err := s.cardRepository.insertCard(txClient, ctx, metadata, tagIDs, importTime, batchOrder)
	if err != nil {
		return nil, err
	}

	tagNames := s.tagRepository.findTagNames(txClient, ctx, tagIDs)
	characterCard.Data.Tags = tagNames

	if err = s.pngRepository.savePng(
		cardHeader.CardID, cardHeader.UpdateTime,
		characterCard,
	); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return cardHeader, nil
}

func (s *Service) UpdateCard(ctx context.Context, cardID scheme.CardID, metadata *models.Metadata, characterCard *png.CharacterCard, checkTime timestamp.Nano) (*scheme.CardHeader, error) {
	if s.client == nil {
		return nil, trace.Err().
			Field(trace.ACTIVITY, "update-card").
			Field(trace.URL, metadata.CardURL).
			Field("cardID", cardID.String()).
			Msg("Could not initialize client")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, trace.Err().
			Field(trace.ACTIVITY, "update-card").
			Field(trace.URL, metadata.CardURL).
			Field("cardID", cardID.String()).
			Msg("Could not initialize transaction")
	}
	txClient := tx.Client()
	defer tx.Rollback()

	if err = s.tagRepository.upsertTags(txClient, ctx, metadata.Tags); err != nil {
		return nil, trace.Err().
			Field(trace.ACTIVITY, "update-card").
			Field(trace.URL, metadata.CardURL).
			Field("cardID", cardID.String()).
			Msg("Could not inset tags")
	}
	tagIDs := slicesx.Map(metadata.Tags, func(tag models.Tag) scheme.TagID {
		return scheme.TagID(tag.Slug)
	})

	cardHeader, err := s.cardRepository.updateCard(txClient, ctx, cardID, metadata, tagIDs,
		scheme.UpdateHeader{
			CheckTime:        checkTime,
			LastUpdateStatus: scheme.UpdateSuccess,
		},
	)
	if err != nil {
		return nil, err
	}

	tagNames := s.tagRepository.findTagNames(txClient, ctx, tagIDs)
	characterCard.Data.Tags = tagNames

	if err = s.pngRepository.savePng(
		cardHeader.CardID,
		cardHeader.UpdateTime,
		characterCard,
	); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return cardHeader, nil
}

func (s *Service) UpdateStatus(ctx context.Context, cardID scheme.CardID, checkTime timestamp.Nano, status scheme.UpdateStatus) error {
	if s.client == nil {
		return trace.Err().
			Field(trace.ACTIVITY, "update-card-status").
			Field("cardID", cardID.String()).
			Msg("UPDATE STATUS - Failed to initialize client")
	}

	return s.cardRepository.updateCardStatus(s.client, ctx, cardID, checkTime, status)
}

func (s *Service) UpdateToLatestExport(ctx context.Context, cardID scheme.CardID, exportTime timestamp.Nano) error {
	return s.cardRepository.updateToLatestExport(s.client, ctx, cardID, exportTime)
}

func (s *Service) ToggleFavorite(ctx context.Context, cardID scheme.CardID) error {
	return s.cardRepository.toggleFavorite(s.client, ctx, cardID)
}

func (s *Service) SetFavorites(ctx context.Context, cardIDs []scheme.CardID, favorite bool) error {
	return s.cardRepository.setFavorites(s.client, ctx, cardIDs, favorite)
}
