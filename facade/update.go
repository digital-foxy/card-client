package facade

import (
	"context"

	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/store/catalog"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/task"
	"github.com/r3dpixel/toolkit/stringsx"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

func (f *Facade) UpdateCards(force bool, rids ...resource.RID) (operation.ID, error) {
	unlock, err := f.beginReadStoreOp()
	if err != nil {
		return operation.EmptyID, err
	}

	handle := f.registry.RegisterUpdate(f.vault.Name)

	go func() {
		defer unlock()
		defer handle.Complete()

		_ = handle.Mutate(func(report *operation.UpdateReport) {
			report.Total = len(rids)
		})

		for _, rid := range rids {
			updateStatus, err := f.updateSingleCard(handle.Context(), rid, handle.TimeStarted(), force)
			f.updateRequestCache.Push(rid, handle.ID())
			if err != nil {
				log.Error().Err(err).
					Str(trace.SERVICE, "facade").
					Str(trace.ACTIVITY, "update-cards").
					Str("cardID", rid.String()).
					Msg("Failed to update card")
			}
			_ = handle.Mutate(func(report *operation.UpdateReport) {
				report.Progress++
				switch updateStatus {
				case resource.SyncSuccess:
					report.NoSuccesses++
				case resource.SyncUnchanged:
					report.NoUnchanges++
				case resource.SyncFailed:
					report.NoFailures++
				}
			})
		}
	}()

	return handle.ID(), nil
}

func (f *Facade) updateSingleCard(ctx context.Context, rid resource.RID, checkTime timestamp.Nano, force bool) (resource.SyncStatus, error) {
	f.tracker.LockItem(rid)
	defer f.tracker.UnlockItem(rid)
	catalog := f.vault.Catalog.WithContext(ctx)
	rec, err := catalog.FindRecord(rid)
	if err != nil {
		return resource.SyncFailed, updateErr("Could not find mini header for card ID", rid, stringsx.Empty, err)
	}

	fetchTask, ok := f.router.TaskOf(rec.NormalizedURL)
	if !ok {
		return resource.SyncFailed, updateErr("Could not find fetcher for URL", rec.ID, rec.DirectURL, nil)
	}

	metadata, err := fetchTask.FetchMetadata()
	if err != nil {
		updateStatus := resource.SyncFailed
		_ = catalog.UpdateSyncData(rid, resource.SyncData{
			SyncTime:   checkTime,
			SyncStatus: updateStatus,
		})
		return updateStatus, updateErr("Could not fetch metadata", rec.ID, rec.DirectURL, err)
	}

	isUpdated := metadata.LatestUpdateTime() > rec.LatestUpdateTime()
	creatorChanged := (metadata.CreatorInfo.Username != rec.Creator.Username) || (metadata.CreatorInfo.Nickname != rec.Creator.Nickname)

	if !force && !isUpdated && !creatorChanged {
		updateStatus := resource.SyncUnchanged
		_ = catalog.UpdateSyncData(rec.ID, resource.SyncData{
			SyncTime:   checkTime,
			SyncStatus: updateStatus,
		})
		return updateStatus, nil
	}

	if err = f.executeFullUpdate(catalog, fetchTask, rec.ID, checkTime); err != nil {
		updateStatus := resource.SyncFailed
		_ = catalog.UpdateSyncData(rec.ID, resource.SyncData{
			SyncTime:   checkTime,
			SyncStatus: updateStatus,
		})
		return updateStatus, err
	}

	return resource.SyncSuccess, nil
}

func (f *Facade) executeFullUpdate(
	catalog catalog.Service,
	fetchTask task.Task,
	rid resource.RID,
	checkTime timestamp.Nano,
) error {
	metadata, card, err := fetchTask.FetchAll()
	if err != nil {
		return updateErr("Could not fetch card", rid, fetchTask.OriginalURL(), err)
	}

	if err = catalog.UpdateCard(rid, metadata, card, checkTime); err != nil {
		return updateErr("Could not save card for ID", rid, fetchTask.OriginalURL(), err)
	}

	return nil
}

func updateErr(msg string, cardID resource.RID, cardURL string, err error) error {
	return trace.Err().
		Wrap(err).
		Msg(msg).
		Field("cardID", cardID).
		Field(trace.URL, cardURL)
}
