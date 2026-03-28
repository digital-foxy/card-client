package facade

import (
	"context"

	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/scheduler"
)

// integrityContext holds all state needed during an integrity check operation.
type integrityContext struct {
	catalog catalog.Service
	handle  operation.Handle
}

// integrityResult holds the outcome of a single record integrity check.
type integrityResult struct {
	Title  string
	Status resource.RecordIntegrity
}

// CheckIntegrity initiates an integrity check for all cards in the vault.
func (s *syncService) CheckIntegrity() (operation.ID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return operation.EmptyID, err
	}

	handle := s.registry.NewOperation(vault.Name, operation.Integrity)
	ctx := &integrityContext{
		catalog: vault.Catalog.WithContext(handle.Context),
		handle:  handle,
	}

	go s.runIntegrityWorker(ctx, unlock)

	return handle.OperationID, nil
}

// runIntegrityWorker executes the integrity check in a background goroutine.
func (s *syncService) runIntegrityWorker(ctx *integrityContext, unlock func()) {
	defer unlock()
	defer s.registry.MarkTerminated(ctx.handle.OperationID)

	checkResult := s.runBasicIntegrityCheck(ctx)
	if !checkResult.basicIntegrity {
		return
	}

	s.processIntegrityRecords(ctx, checkResult.rids)
	_, _ = ctx.catalog.CleanupCreators()
}

// integrityCheckResult holds the outcome of the integrity check setup.
type integrityCheckResult struct {
	rids           []resource.RID
	basicIntegrity bool
}

// runBasicIntegrityCheck performs the initial integrity check and returns affected RIDs.
func (s *syncService) runBasicIntegrityCheck(ctx *integrityContext) integrityCheckResult {
	rids, integrity := ctx.catalog.BasicIntegrity()
	return integrityCheckResult{
		rids:           rids,
		basicIntegrity: integrity,
	}
}

// processIntegrityRecords fetches and processes all records for integrity checking.
func (s *syncService) processIntegrityRecords(ctx *integrityContext, rids []resource.RID) {
	s.setIntegrityTotal(ctx.handle.OperationID, len(rids))

	records, err := ctx.catalog.FindRecords(rids...)
	if err != nil {
		return
	}

	s.checkAllRecordIntegrity(ctx, records)
}

// setIntegrityTotal sets the total count for the integrity operation report.
func (s *syncService) setIntegrityTotal(opID operation.ID, total int) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Total = total
	})
}

// checkAllRecordIntegrity processes all records in parallel using the scheduler.
func (s *syncService) checkAllRecordIntegrity(ctx *integrityContext, records resource.Box[resource.Record]) {
	scheduler.Exec(scheduler.FromSlice(records.Items), scheduler.Options[resource.Record]{
		Context:     ctx.handle.Context,
		Parallelism: s.workers,
		Handler: func(_ context.Context, rec resource.Record) {
			result := s.processSingleIntegrity(ctx, &rec)
			s.updateIntegrityProgress(ctx.handle.OperationID, result)
		},
	})
}

// processSingleIntegrity handles the integrity check of a single record.
func (s *syncService) processSingleIntegrity(ctx *integrityContext, rec *resource.Record) integrityResult {
	s.tracker.LockItem(rec.ID)
	defer s.tracker.UnlockItem(rec.ID)

	status := ctx.catalog.FixRecordIntegrity(rec)
	return integrityResult{
		Title:  rec.Title,
		Status: status,
	}
}

// updateIntegrityProgress updates the operation report with the result of an integrity check.
func (s *syncService) updateIntegrityProgress(opID operation.ID, result integrityResult) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Progress++
		switch result.Status {
		case resource.OK:
			report.NoSuccesses++
		case resource.BROKEN:
			report.AuxData = append(report.AuxData, result.Title)
			report.NoFailures++
		case resource.FIXED:
			report.NoFixes++
		}
	})
}
