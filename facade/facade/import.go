package facade

import (
	"context"
	"strings"

	"github.com/r3dpixel/card-fetcher/task"
	"github.com/r3dpixel/toolkit/structx"
	"github.com/r3dpixel/toolkit/symbols"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

func (s *Service) ImportURLs(rawURLs string) (operation.ID, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return operation.EmptyID, err
	} else {
		defer unlock()
	}

	handle := s.registry.RegisterImport(s.storeService.VaultName())
	defer handle.Complete()

	urls := strings.Split(rawURLs, symbols.NewLine)
	_ = handle.Mutate(func(report *operation.ImportReport) {
		report.Total = len(urls)
	})

	tasks := s.routerService.TaskMapOf(urls...)
	foundSlice := s.storeService.FindURLs(handle.Context(), tasks.ValidURLs)
	foundMap := make(map[string]struct{})
	for _, existingURL := range foundSlice {
		foundMap[existingURL] = structx.Empty
	}
	_ = handle.Mutate(func(report *operation.ImportReport) {
		report.Progress = len(tasks.InvalidURLs)
		report.NoValidURLs = len(tasks.ValidURLs)
		report.NoInvalidURLs = len(tasks.InvalidURLs)
	})

	for index, toFetchURL := range tasks.ValidURLs {
		fetchTask := tasks.Tasks[toFetchURL]
		s.handleImportTask(handle, fetchTask, foundMap, index)
	}

	return handle.ID(), nil
}

func (s *Service) handleImportTask(handle operation.Handle[*operation.ImportReport], fetchTask task.Task, foundMap map[string]struct{}, batchOrder int) {
	var importStatus scheme.ImportStatus
	var err error

	if _, exists := foundMap[fetchTask.NormalizedURL()]; exists {
		importStatus = scheme.ImportDuplicate
		err = nil
	} else {
		importStatus, err = s.importSingleCard(handle.Context(), fetchTask, handle.TimeStarted(), batchOrder)
		if err != nil {
			log.Error().Err(err).
				Str(trace.SERVICE, "facade").
				Str(trace.ACTIVITY, "import-cards").
				Str(trace.URL, fetchTask.NormalizedURL()).
				Msg("Could not import card")
		}
	}

	_ = handle.Mutate(func(report *operation.ImportReport) {
		report.Progress++
		switch importStatus {
		case scheme.ImportSuccess:
			report.NoSuccesses++
		case scheme.ImportFailed:
			report.NoFailures++
		case scheme.ImportDuplicate:
			report.NoDuplicates++
		}
	})
}

func (s *Service) importSingleCard(ctx context.Context, fetchTask task.Task, importTime timestamp.Nano, batchOrder int) (scheme.ImportStatus, error) {
	metadata, card, err := fetchTask.FetchAll()
	if err != nil {
		return scheme.ImportFailed, err
	}

	if _, err = s.storeService.InsertCard(ctx, metadata, card, importTime, batchOrder); err != nil {
		if ent.IsConstraintError(err) {
			return scheme.ImportDuplicate, nil
		}
		return scheme.ImportFailed, err
	}
	return scheme.ImportSuccess, nil
}
