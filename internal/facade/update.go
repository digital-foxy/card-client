package facade

import (
	"context"

	"github.com/r3dpixel/card-client/services/operation"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/task"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

func (s *Service) UpdateCards(force bool, cardIDs ...scheme.CardID) (operation.ID, error) {
	unlock, err := s.beginReadStoreOp()
	if err != nil {
		return operation.EmptyID, err
	}

	handle := s.registry.RegisterUpdate(s.storeService.VaultName())

	go func() {
		defer unlock()
		defer handle.Complete()

		_ = handle.Mutate(func(report *operation.UpdateReport) {
			report.Total = len(cardIDs)
		})

		for _, cardID := range cardIDs {
			updateStatus, err := s.updateSingleCard(handle.Context(), cardID, handle.TimeStarted(), force)
			s.updateRequestCache.Push(cardID, handle.ID())
			if err != nil {
				log.Error().Err(err)
			}
			_ = handle.Mutate(func(report *operation.UpdateReport) {
				report.Progress++
				switch updateStatus {
				case scheme.UpdateSuccess:
					report.NoSuccesses++
				case scheme.UpdateUnchanged:
					report.NoUnchanges++
				case scheme.UpdateFailed:
					report.NoFailures++
				}
			})
		}
	}()

	return handle.ID(), nil
}

func (s *Service) updateSingleCard(ctx context.Context, cardID scheme.CardID, checkTime timestamp.Nano, force bool) (scheme.UpdateStatus, error) {
	s.trackerService.LockItem(cardID)
	defer s.trackerService.UnlockItem(cardID)
	miniHeader, err := s.storeService.FindMiniHeader(ctx, cardID)
	if err != nil {
		return scheme.UpdateFailed, updateErr("Could not find mini header for card ID", cardID, miniHeader.CardURL, err)
	}

	fetchTask, ok := s.routerService.TaskOf(miniHeader.CardURL)
	if !ok {
		return scheme.UpdateFailed, updateErr("Could not find fetcher for CardURL", miniHeader.CardID, miniHeader.CardURL, nil)
	}

	metadata, err := fetchTask.FetchMetadata()
	if err != nil {
		updateStatus := scheme.UpdateFailed
		_ = s.storeService.UpdateStatus(ctx, miniHeader.CardID, checkTime, updateStatus)
		return updateStatus, updateErr("Could not fetch metadata", miniHeader.CardID, miniHeader.CardURL, err)
	}

	isUpdated := metadata.LatestUpdateTime() > max(miniHeader.UpdateTime, miniHeader.BookUpdateTime)
	creatorChanged := metadata.Creator != miniHeader.Creator

	if !force && !isUpdated && !creatorChanged {
		updateStatus := scheme.UpdateUnchanged
		_ = s.storeService.UpdateStatus(ctx, miniHeader.CardID, checkTime, updateStatus)
		return updateStatus, nil
	}

	if err = s.executeFullUpdate(ctx, fetchTask, miniHeader.CardID, checkTime); err != nil {
		updateStatus := scheme.UpdateFailed
		_ = s.storeService.UpdateStatus(ctx, miniHeader.CardID, checkTime, updateStatus)
		return updateStatus, err
	}

	return scheme.UpdateSuccess, nil
}

func (s *Service) executeFullUpdate(
	ctx context.Context,
	fetchTask task.Task,
	cardID scheme.CardID,
	checkTime timestamp.Nano,
) error {
	metadata, card, err := fetchTask.FetchAll()
	if err != nil {
		return updateErr("Could not fetch card", cardID, metadata.CardURL, err)
	}

	if _, err = s.storeService.UpdateCard(ctx, cardID, metadata, card, checkTime); err != nil {
		return updateErr("Could not save card for ID", cardID, metadata.CardURL, err)
	}

	return nil
}

func updateErr(msg string, cardID scheme.CardID, cardURL string, err error) error {
	return trace.Err().
		Wrap(err).
		Msg(msg).
		Field("cardID", cardID).
		Field(trace.URL, cardURL)
}
