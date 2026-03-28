package facade

import (
	"context"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/task"
	"github.com/digital-foxy/toolkit/scheduler"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/digital-foxy/toolkit/trace"
	"github.com/rs/zerolog/log"
)

// updateResult holds the outcome of a single card update operation.
type updateResult struct {
	Title  string
	Status resource.SyncStatus
}

// updateContext holds all state needed during an update operation.
type updateContext struct {
	catalog   catalog.Service
	vault     library.Vault
	handle    operation.Handle
	checkTime timestamp.Nano
	force     bool
}

// cardUpdateParams groups parameters for updating a single card.
type cardUpdateParams struct {
	rid       resource.RID
	record    *resource.Record
	fetchTask task.Task
}

// metadataCheckResult holds the result of checking if a card needs updating.
type metadataCheckResult struct {
	needsUpdate bool
	metadata    *models.Metadata
	recordTitle string
}

// UpdateCards initiates async updates for the specified cards.
func (s *syncService) UpdateCards(force bool, rids ...resource.RID) (operation.ID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return operation.EmptyID, err
	}

	handle := s.registry.NewOperation(vault.Name, operation.Update)
	ctx := s.buildUpdateContext(vault, handle, force)

	go s.runUpdateWorker(ctx, unlock, rids)

	return handle.OperationID, nil
}

// buildUpdateContext creates an update context from the vault and operation handle.
func (s *syncService) buildUpdateContext(vault library.Vault, handle operation.Handle, force bool) *updateContext {
	return &updateContext{
		catalog:   vault.Catalog.WithContext(handle.Context),
		vault:     vault,
		handle:    handle,
		checkTime: handle.TimeStarted,
		force:     force,
	}
}

// runUpdateWorker executes the update operation in a background goroutine.
func (s *syncService) runUpdateWorker(ctx *updateContext, unlock func(), rids []resource.RID) {
	defer unlock()

	s.setUpdateTotal(ctx.handle.OperationID, len(rids))
	s.processAllCards(ctx, rids)

	s.registry.MarkTerminated(ctx.handle.OperationID)
}

// setUpdateTotal sets the total count for the update operation report.
func (s *syncService) setUpdateTotal(opID operation.ID, total int) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Total = total
	})
}

// processAllCards processes all RIDs in parallel using the scheduler.
func (s *syncService) processAllCards(ctx *updateContext, rids []resource.RID) {
	scheduler.Exec(scheduler.FromSlice(rids), scheduler.Options[resource.RID]{
		Context:     ctx.handle.Context,
		Parallelism: s.workers,
		Handler: func(_ context.Context, rid resource.RID) {
			result := s.processSingleCard(ctx, rid)
			s.updateUpdateProgress(ctx.handle.OperationID, result)
		},
	})
}

// processSingleCard handles the update of a single card and pushes to cache.
func (s *syncService) processSingleCard(ctx *updateContext, rid resource.RID) updateResult {
	result, err := s.updateSingleCard(ctx, rid)
	s.cache.Push(rid, ctx.handle.OperationID)

	if err != nil {
		s.logUpdateError(rid, err)
	}
	return result
}

// updateSingleCard performs the full update workflow for a single card.
func (s *syncService) updateSingleCard(ctx *updateContext, rid resource.RID) (updateResult, error) {
	s.tracker.LockItem(rid)
	defer s.tracker.UnlockItem(rid)

	rec, err := ctx.catalog.FindRecord(rid)
	if err != nil {
		return s.handleRecordNotFound(rec, rid, err)
	}

	fetchTask, ok := s.router.TaskOf(rec.NormalizedURL)
	if !ok {
		return s.handleFetcherNotFound(rec)
	}

	params := cardUpdateParams{
		rid:       rid,
		record:    rec,
		fetchTask: fetchTask,
	}
	return s.executeCardUpdate(ctx, params)
}

// handleRecordNotFound handles the case when a record cannot be found.
func (s *syncService) handleRecordNotFound(rec *resource.Record, rid resource.RID, err error) (updateResult, error) {
	wrappedErr := updateErr("Could not find record for card ID", rid, "", err)
	return updateResult{
		Title:  "",
		Status: resource.SyncFailed,
	}, wrappedErr
}

// handleFetcherNotFound handles the case when no fetcher is available for a URL.
func (s *syncService) handleFetcherNotFound(rec *resource.Record) (updateResult, error) {
	err := updateErr("Could not find fetcher for URL", rec.ID, rec.DirectURL, nil)
	return updateResult{
		Title:  rec.Title,
		Status: resource.SyncFailed,
	}, err
}

// executeCardUpdate performs the actual update logic for a card.
func (s *syncService) executeCardUpdate(ctx *updateContext, params cardUpdateParams) (updateResult, error) {
	checkResult, err := s.checkMetadata(ctx, params)
	if err != nil {
		return checkResult.toFailedResult(), err
	}

	if !checkResult.needsUpdate {
		return s.markUnchanged(ctx, params.rid, checkResult.metadata.Title)
	}

	return s.performFullUpdate(ctx, params, checkResult.metadata.Title)
}

// checkMetadata fetches and evaluates whether a card needs updating.
func (s *syncService) checkMetadata(ctx *updateContext, params cardUpdateParams) (metadataCheckResult, error) {
	metadata, err := params.fetchTask.FetchMetadata()
	if err != nil {
		s.setSyncStatus(ctx.catalog, params.rid, ctx.checkTime, resource.SyncFailed)
		wrappedErr := updateErr("Could not fetch metadata", params.record.ID, params.record.DirectURL, err)
		return metadataCheckResult{metadata: metadata, recordTitle: params.record.Title}, wrappedErr
	}

	needsUpdate := s.determineIfUpdateNeeded(ctx.force, metadata, params.record)
	return metadataCheckResult{
		needsUpdate: needsUpdate,
		metadata:    metadata,
	}, nil
}

// determineIfUpdateNeeded checks if a card requires updating based on timestamps and creator info.
func (s *syncService) determineIfUpdateNeeded(force bool, metadata *models.Metadata, rec *resource.Record) bool {
	if force {
		return true
	}

	isUpdated := metadata.LatestUpdateTime() > rec.LatestUpdateTime()
	creatorChanged := s.hasCreatorChanged(metadata, rec)

	return isUpdated || creatorChanged
}

// hasCreatorChanged checks if the creator information has changed.
func (s *syncService) hasCreatorChanged(metadata *models.Metadata, rec *resource.Record) bool {
	usernameChanged := metadata.CreatorInfo.Username != rec.Creator.Username
	nicknameChanged := metadata.CreatorInfo.Nickname != rec.Creator.Nickname
	return usernameChanged || nicknameChanged
}

// markUnchanged marks a card as unchanged and returns the appropriate result.
func (s *syncService) markUnchanged(ctx *updateContext, rid resource.RID, title string) (updateResult, error) {
	s.setSyncStatus(ctx.catalog, rid, ctx.checkTime, resource.SyncUnchanged)
	return updateResult{
		Title:  title,
		Status: resource.SyncUnchanged,
	}, nil
}

// performFullUpdate executes a full card update including fetching and saving.
func (s *syncService) performFullUpdate(ctx *updateContext, params cardUpdateParams, title string) (updateResult, error) {
	if err := s.executeFullUpdate(ctx.catalog, params.fetchTask, params.rid, ctx.checkTime); err != nil {
		s.setSyncStatus(ctx.catalog, params.rid, ctx.checkTime, resource.SyncFailed)
		return updateResult{
			Title:  title,
			Status: resource.SyncFailed,
		}, err
	}

	return updateResult{
		Title:  title,
		Status: resource.SyncSuccess,
	}, nil
}

// executeFullUpdate fetches all card data and saves it to the catalog.
func (s *syncService) executeFullUpdate(
	ctg catalog.Service,
	fetchTask task.Task,
	rid resource.RID,
	checkTime timestamp.Nano,
) error {
	metadata, card, err := fetchTask.FetchAll()
	if err != nil {
		return updateErr("Could not fetch card", rid, fetchTask.OriginalURL(), err)
	}

	if _, err = ctg.SaveCard(metadata, card, checkTime); err != nil {
		return updateErr("Could not save card for ID", rid, fetchTask.OriginalURL(), err)
	}

	return nil
}

// setSyncStatus updates the sync status for a card in the catalog.
func (s *syncService) setSyncStatus(ctg catalog.Service, rid resource.RID, checkTime timestamp.Nano, status resource.SyncStatus) {
	_ = ctg.UpdateSyncData(rid, resource.SyncData{
		SyncTime:   checkTime,
		SyncStatus: status,
	})
}

// toFailedResult converts a metadata check result to a failed update result.
func (m metadataCheckResult) toFailedResult() updateResult {
	return updateResult{
		Title:  m.recordTitle,
		Status: resource.SyncFailed,
	}
}

// logUpdateError logs an error that occurred during card update.
func (s *syncService) logUpdateError(rid resource.RID, err error) {
	log.Error().Err(err).
		Str(trace.SERVICE, "sync").
		Str(trace.ACTIVITY, "update-cards").
		Str("cardID", rid.String()).
		Msg("Failed to update card")
}

// updateUpdateProgress updates the operation report with the result of an update attempt.
func (s *syncService) updateUpdateProgress(opID operation.ID, result updateResult) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Progress++
		switch result.Status {
		case resource.SyncSuccess:
			report.NoSuccesses++
		case resource.SyncUnchanged:
			report.NoUnchanges++
		case resource.SyncFailed:
			report.AuxData = append(report.AuxData, result.Title)
			report.NoFailures++
		}
	})
}

// updateErr creates a traced error for update operations.
func updateErr(msg string, cardID resource.RID, cardURL string, err error) error {
	return trace.Error().
		Wrap(err).
		Msg(msg).
		Field("cardID", cardID).
		Field(trace.URL, cardURL)
}
