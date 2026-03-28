package facade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-client/store/templater"
	"github.com/digital-foxy/toolkit/filex"
	"github.com/digital-foxy/toolkit/jsonx"
	"github.com/digital-foxy/toolkit/scheduler"
)

// ExportVault exports all cards (the latest version) to a vault-named folder.
// Creates files with a pattern: {rid}_{template}.png
// Creates an index.json with map[rid]resource.Record.
func (s *exportService) ExportVault() (operation.ID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return operation.EmptyID, err
	}

	handle := s.registry.NewOperation(vault.Name, operation.ExportVault)

	go s.runVaultExportWorker(vault, handle, unlock)

	return handle.OperationID, nil
}

type vaultExportContext struct {
	cat        catalog.Service
	handle     operation.Handle
	exportPath string
	compiled   *templater.CompiledTemplate
	indexMu    sync.Mutex
	index      map[resource.RID]resource.Record
}

func (s *exportService) runVaultExportWorker(vault library.Vault, handle operation.Handle, unlock func()) {
	defer unlock()
	defer s.registry.MarkTerminated(handle.OperationID)

	cat := vault.Catalog.WithContext(handle.Context)
	exportRoot := s.preferences.GetString(preferences.ExportPathKey)
	exportPath := filex.NextAvailablePath(filepath.Join(exportRoot, string(vault.Name)))

	if err := os.MkdirAll(exportPath, filex.DirectoryPermission); err != nil {
		return
	}

	rids, err := cat.FindPagedRIDs(resource.Filter{}, 0, -1)
	if err != nil {
		return
	}

	records, err := cat.FindRecords(rids...)
	if err != nil {
		return
	}

	_ = s.registry.MutateReport(handle.OperationID, func(report *operation.Report) {
		report.Total = len(records.Items)
	})

	ctx := &vaultExportContext{
		cat:        cat,
		handle:     handle,
		exportPath: exportPath,
		compiled:   s.getCompiledTemplate(),
		index:      make(map[resource.RID]resource.Record, len(records.Items)),
	}

	scheduler.Exec(scheduler.FromSlice(records.Items), scheduler.Options[resource.Record]{
		Context:     handle.Context,
		Parallelism: s.workers,
		Handler: func(_ context.Context, rec resource.Record) {
			s.exportVaultCard(ctx, rec)
		},
	})

	_ = jsonx.ToFile(ctx.index, filepath.Join(exportPath, "index.json"))
}

func (s *exportService) exportVaultCard(ctx *vaultExportContext, rec resource.Record) {
	cardBytes, err := ctx.cat.GetCardBytes(rec.ID, rec.UpdateTime)
	if err != nil {
		s.updateVaultExportProgress(ctx.handle.OperationID, rec.Title, false)
		return
	}

	filename := fmt.Sprintf("%d_%s", rec.ID, filex.SanitizePath(ctx.compiled.Execute(&rec)))
	if err := os.WriteFile(filepath.Join(ctx.exportPath, filename), cardBytes, filex.FilePermission); err != nil {
		s.updateVaultExportProgress(ctx.handle.OperationID, rec.Title, false)
		return
	}

	ctx.indexMu.Lock()
	ctx.index[rec.ID] = rec
	ctx.indexMu.Unlock()

	s.updateVaultExportProgress(ctx.handle.OperationID, rec.Title, true)
}

func (s *exportService) updateVaultExportProgress(opID operation.ID, title string, success bool) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Progress++
		if success {
			report.NoSuccesses++
		} else {
			report.AuxData = append(report.AuxData, title)
			report.NoFailures++
		}
	})
}
