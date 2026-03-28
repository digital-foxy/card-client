package facade

import (
	"context"
	"strings"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/record/erecord/ent"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/task"
	"github.com/digital-foxy/toolkit/scheduler"
	"github.com/digital-foxy/toolkit/structx"
	"github.com/digital-foxy/toolkit/symbols"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/digital-foxy/toolkit/trace"
	"github.com/rs/zerolog/log"
)

// importContext holds all state needed during an import operation.
type importContext struct {
	catalog     catalog.Service
	handle      operation.Handle
	existingMap map[string]struct{}
}

// importTaskParams groups parameters for processing a single import task.
type importTaskParams struct {
	task       task.Task
	batchOrder int
}

// importCardParams groups parameters for inserting a single card.
type importCardParams struct {
	task       task.Task
	importTime timestamp.Nano
	batchOrder int
}

// ImportURLs initiates the import of cards from a newline-separated list of URLs.
func (s *syncService) ImportURLs(rawURLs string) (operation.ID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return operation.EmptyID, err
	}
	defer unlock()

	handle := s.registry.NewOperation(vault.Name, operation.Import)

	ctx, err := s.buildImportContext(vault, handle, rawURLs)
	if err != nil {
		return operation.EmptyID, err
	}

	s.processAllImportTasks(ctx, rawURLs)
	s.registry.MarkTerminated(handle.OperationID)
	return handle.OperationID, nil
}

// buildImportContext prepares the import context by parsing URLs and finding existing cards.
func (s *syncService) buildImportContext(vault library.Vault, handle operation.Handle, rawURLs string) (*importContext, error) {
	urls := strings.Split(rawURLs, symbols.NewLine)
	s.setImportTotal(handle.OperationID, len(urls))

	ctg := vault.Catalog.WithContext(handle.Context)
	tasks := s.router.TaskMapOf(urls...)

	existingMap, err := s.buildExistingURLMap(ctg, tasks.ValidURLs)
	if err != nil {
		return nil, err
	}

	s.updateImportURLCounts(handle.OperationID, len(tasks.ValidURLs), len(tasks.InvalidURLs))

	return &importContext{
		catalog:     ctg,
		handle:      handle,
		existingMap: existingMap,
	}, nil
}

// setImportTotal sets the total count for the import operation report.
func (s *syncService) setImportTotal(opID operation.ID, total int) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Total = total
	})
}

// buildExistingURLMap creates a lookup map of URLs that already exist in the catalog.
func (s *syncService) buildExistingURLMap(ctg catalog.Service, validURLs []string) (map[string]struct{}, error) {
	foundSlice, err := ctg.FindURLs(validURLs...)
	if err != nil {
		return nil, err
	}

	existingMap := make(map[string]struct{}, len(foundSlice))
	for _, existingURL := range foundSlice {
		existingMap[existingURL] = structx.Empty
	}
	return existingMap, nil
}

// updateImportURLCounts updates the report with URL validation counts.
func (s *syncService) updateImportURLCounts(opID operation.ID, validCount, invalidCount int) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Progress = invalidCount
		report.NoValidURLs = validCount
		report.NoInvalidURLs = invalidCount
	})
}

// processAllImportTasks iterates through all valid URLs and processes each import task.
func (s *syncService) processAllImportTasks(ctx *importContext, rawURLs string) {
	urls := strings.Split(rawURLs, symbols.NewLine)
	tasks := s.router.TaskMapOf(urls...)

	// Build slice of import task params
	paramsList := make([]importTaskParams, len(tasks.ValidURLs))
	for index, toFetchURL := range tasks.ValidURLs {
		paramsList[index] = importTaskParams{
			task:       tasks.Tasks[toFetchURL],
			batchOrder: index,
		}
	}

	// Process tasks in parallel using scheduler
	scheduler.Exec(scheduler.FromSlice(paramsList), scheduler.Options[importTaskParams]{
		Context:     ctx.handle.Context,
		Parallelism: s.workers,
		Handler: func(_ context.Context, params importTaskParams) {
			s.processImportTask(ctx, params)
		},
	})
}

// processImportTask handles a single import task, checking for duplicates and importing if needed.
func (s *syncService) processImportTask(ctx *importContext, params importTaskParams) {
	status := s.determineImportStatus(ctx, params)
	s.updateImportProgress(ctx.handle.OperationID, status, params.task.OriginalURL())
}

// determineImportStatus checks if the URL exists and imports if it doesn't.
func (s *syncService) determineImportStatus(ctx *importContext, params importTaskParams) resource.ImportStatus {
	if s.isURLAlreadyImported(ctx.existingMap, params.task.NormalizedURL()) {
		return resource.ImportDuplicate
	}

	cardParams := importCardParams{
		task:       params.task,
		importTime: ctx.handle.TimeStarted,
		batchOrder: params.batchOrder,
	}

	status, err := s.importSingleCard(ctx.catalog, cardParams)
	if err != nil {
		s.logImportError(params.task.NormalizedURL(), err)
	}
	return status
}

// isURLAlreadyImported checks if a URL already exists in the import map.
func (s *syncService) isURLAlreadyImported(existingMap map[string]struct{}, normalizedURL string) bool {
	_, exists := existingMap[normalizedURL]
	return exists
}

// importSingleCard fetches and inserts a single card into the catalog.
func (s *syncService) importSingleCard(ctg catalog.Service, params importCardParams) (resource.ImportStatus, error) {
	metadata, card, err := params.task.FetchAll()
	if err != nil {
		return resource.ImportFailed, err
	}

	if _, err = ctg.SaveCard(metadata, card, params.importTime, params.batchOrder); err != nil {
		return s.handleInsertError(err)
	}
	return resource.ImportSuccess, nil
}

// handleInsertError determines the appropriate status based on the insert error type.
func (s *syncService) handleInsertError(err error) (resource.ImportStatus, error) {
	if ent.IsConstraintError(err) {
		return resource.ImportDuplicate, nil
	}
	return resource.ImportFailed, err
}

// logImportError logs an error that occurred during card import.
func (s *syncService) logImportError(url string, err error) {
	log.Error().Err(err).
		Str(trace.SERVICE, "sync").
		Str(trace.ACTIVITY, "import-cards").
		Str(trace.URL, url).
		Msg("Could not import card")
}

// updateImportProgress updates the operation report with the result of an import attempt.
func (s *syncService) updateImportProgress(opID operation.ID, status resource.ImportStatus, originalURL string) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Progress++
		switch status {
		case resource.ImportSuccess:
			report.NoSuccesses++
		case resource.ImportFailed:
			report.AuxData = append(report.AuxData, originalURL)
			report.NoFailures++
		case resource.ImportDuplicate:
			report.NoDuplicates++
		}
	})
}
