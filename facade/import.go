package facade

import (
	"strings"

	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/store/catalog"
	"github.com/r3dpixel/card-client/store/record/erecord/ent"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/task"
	"github.com/r3dpixel/toolkit/structx"
	"github.com/r3dpixel/toolkit/symbols"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
)

func (f *Facade) ImportURLs(rawURLs string) (operation.ID, error) {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return operation.EmptyID, err
	} else {
		defer unlock()
	}

	handle := f.registry.RegisterImport(f.vault.Name)
	defer handle.Complete()

	urls := strings.Split(rawURLs, symbols.NewLine)
	_ = handle.Mutate(func(report *operation.ImportReport) {
		report.Total = len(urls)
	})

	tasks := f.router.TaskMapOf(urls...)
	catalog := f.vault.Catalog.WithContext(handle.Context())
	foundSlice, err := catalog.FindURLs(tasks.ValidURLs...)
	if err != nil {
		return operation.EmptyID, err
	}

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
		f.handleImportTask(catalog, handle, fetchTask, foundMap, index)
	}

	return handle.ID(), nil
}

func (f *Facade) handleImportTask(catalog catalog.Service, handle operation.Handle[*operation.ImportReport], fetchTask task.Task, foundMap map[string]struct{}, batchOrder int) {
	var importStatus resource.ImportStatus
	var err error

	if _, exists := foundMap[fetchTask.NormalizedURL()]; exists {
		importStatus = resource.ImportDuplicate
		err = nil
	} else {
		importStatus, err = f.importSingleCard(catalog, fetchTask, handle.TimeStarted(), batchOrder)
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
		case resource.ImportSuccess:
			report.NoSuccesses++
		case resource.ImportFailed:
			report.NoFailures++
		case resource.ImportDuplicate:
			report.NoDuplicates++
		}
	})
}

func (f *Facade) importSingleCard(catalog catalog.Service, fetchTask task.Task, importTime timestamp.Nano, batchOrder int) (resource.ImportStatus, error) {
	metadata, card, err := fetchTask.FetchAll()
	if err != nil {
		return resource.ImportFailed, err
	}

	if err = catalog.InsertCard(metadata, card, resource.ImportData{
		ImportTime:  importTime,
		ImportIndex: batchOrder,
	}); err != nil {
		if ent.IsConstraintError(err) {
			return resource.ImportDuplicate, nil
		}
		return resource.ImportFailed, err
	}
	return resource.ImportSuccess, nil
}
