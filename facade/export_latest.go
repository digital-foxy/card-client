package facade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/preferences"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-client/store/templater"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/filex"
	"github.com/digital-foxy/toolkit/scheduler"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/digital-foxy/toolkit/trace"
)

// exportContext holds all state needed during an export operation.
type exportContext struct {
	catalog          catalog.Service
	handle           operation.Handle
	compiledTemplate *templater.CompiledTemplate
	exportPath       string
	maxExportSize    int
}

// exportResult holds the outcome of a single card export operation.
type exportResult struct {
	Identifier string
	Success    bool
}

// ExportLatestCards initiates the export of the specified cards.
func (s *exportService) ExportLatestCards(rids ...resource.RID) (operation.ID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return operation.EmptyID, err
	}

	handle := s.registry.NewOperation(vault.Name, operation.Export)
	ctx := s.buildExportContext(vault, handle)

	go s.runExportWorker(ctx, unlock, rids)

	return handle.OperationID, nil
}

// buildExportContext creates an export context from the vault and operation handle.
func (s *exportService) buildExportContext(vault library.Vault, handle operation.Handle) *exportContext {
	return &exportContext{
		catalog:          vault.Catalog.WithContext(handle.Context),
		handle:           handle,
		compiledTemplate: s.getCompiledTemplate(),
		exportPath:       s.preferences.GetString(preferences.ExportPathKey),
		maxExportSize:    s.preferences.GetInt(preferences.MaxExportSizeKey),
	}
}

// runExportWorker executes the export operation in a background goroutine.
func (s *exportService) runExportWorker(ctx *exportContext, unlock func(), rids []resource.RID) {
	defer unlock()

	s.setExportTotal(ctx.handle.OperationID, len(rids))
	s.processAllExports(ctx, rids)

	s.registry.MarkTerminated(ctx.handle.OperationID)
}

// setExportTotal sets the total count for the export operation report.
func (s *exportService) setExportTotal(opID operation.ID, total int) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Total = total
	})
}

// processAllExports processes all RIDs in parallel using the scheduler.
func (s *exportService) processAllExports(ctx *exportContext, rids []resource.RID) {
	scheduler.Exec(scheduler.FromSlice(rids), scheduler.Options[resource.RID]{
		Context:     ctx.handle.Context,
		Parallelism: s.workers,
		Handler: func(_ context.Context, rid resource.RID) {
			result := s.processSingleExport(ctx, rid)
			s.updateExportProgress(ctx.handle.OperationID, result)
		},
	})
}

// processSingleExport handles the export of a single card and pushes to cache.
func (s *exportService) processSingleExport(ctx *exportContext, rid resource.RID) exportResult {
	identifier, err := s.exportCard(ctx, rid)
	s.cache.Push(rid, ctx.handle.OperationID)

	return exportResult{
		Identifier: identifier,
		Success:    err == nil,
	}
}

// updateExportProgress updates the operation report with the result of an export attempt.
func (s *exportService) updateExportProgress(opID operation.ID, result exportResult) {
	_ = s.registry.MutateReport(opID, func(report *operation.Report) {
		report.Progress++
		if result.Success {
			report.NoSuccesses++
		} else {
			report.AuxData = append(report.AuxData, result.Identifier)
			report.NoFailures++
		}
	})
}

// exportCard exports a single card to the configured export path.
func (s *exportService) exportCard(ctx *exportContext, rid resource.RID) (string, error) {
	s.tracker.LockItem(rid)
	defer s.tracker.UnlockItem(rid)

	rec, err := ctx.catalog.FindRecord(rid)
	if err != nil {
		return "", err
	}

	identifier := s.buildIdentifier(rec)
	dstPath := s.buildExportPath(ctx, rec)

	if err := s.writeCardToFile(ctx, rid, rec, dstPath); err != nil {
		return identifier, err
	}

	s.markExported(ctx.catalog, rid, rec.UpdateTime)
	return identifier, nil
}

// buildIdentifier creates a human-readable identifier for the card.
func (s *exportService) buildIdentifier(rec *resource.Record) string {
	return fmt.Sprintf("%s - %s", rec.Title, rec.DirectURL)
}

// buildExportPath constructs the destination file path for the export.
func (s *exportService) buildExportPath(ctx *exportContext, rec *resource.Record) string {
	return filepath.Join(ctx.exportPath, filex.SanitizePath(ctx.compiledTemplate.Execute(rec)))
}

// writeCardToFile writes the card image to the destination path, scaling if needed.
func (s *exportService) writeCardToFile(ctx *exportContext, rid resource.RID, rec *resource.Record, dstPath string) error {
	cardBytes, err := ctx.catalog.GetCardBytes(rid, rec.UpdateTime)
	if err != nil {
		return err
	}

	if s.needsScaling(cardBytes, ctx.maxExportSize) {
		return s.writeScaledCard(cardBytes, dstPath, ctx.maxExportSize)
	}

	return s.writeOriginalCard(cardBytes, dstPath, rec)
}

// needsScaling checks if the card image exceeds the maximum export size.
func (s *exportService) needsScaling(cardBytes []byte, maxSize int) bool {
	processor := png.FromBytes(cardBytes)
	width, height := processor.ImageSize()
	return width > maxSize || height > maxSize
}

// writeScaledCard scales down the card and writes it to the destination.
func (s *exportService) writeScaledCard(cardBytes []byte, dstPath string, maxSize int) error {
	processor := png.FromBytes(cardBytes)
	rawCard, err := processor.Get()
	if err != nil {
		return err
	}

	if err := rawCard.ScaleDown(maxSize); err != nil {
		return err
	}

	return rawCard.ToFile(dstPath)
}

// writeOriginalCard writes the card bytes directly to the destination.
func (s *exportService) writeOriginalCard(cardBytes []byte, dstPath string, rec *resource.Record) error {
	if err := os.WriteFile(dstPath, cardBytes, filex.FilePermission); err != nil {
		return trace.Error().Wrap(err).
			Field(trace.SOURCE, rec.InfoData.Source).
			Field("cardID", rec.ID).
			Msg("Could not copy card for export")
	}
	return nil
}

// markExported updates the export data for the card in the catalog.
func (s *exportService) markExported(ctg catalog.Service, rid resource.RID, version timestamp.Nano) {
	_ = ctg.UpdateExportData(rid, resource.ExportData{
		ExportTime:      timestamp.NowNano(),
		ExportedVersion: version,
	})
}
