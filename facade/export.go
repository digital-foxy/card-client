package facade

import (
	"context"
	"os"
	"path/filepath"

	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/preferences"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
)

func (f *Facade) ExportLatestCards(rids ...resource.RID) (operation.ID, error) {
	unlock, err := f.beginReadStoreOp()
	if err != nil {
		return operation.EmptyID, err
	}
	handle := f.registry.RegisterExport(f.vault.Name)
	maxExportSize := f.preferences.GetInt(preferences.MaxExportSizeKey)

	go func() {
		defer unlock()
		defer handle.Complete()

		exportPath := f.preferences.GetString(preferences.ExportPathKey)

		f.builderMutex.Lock()
		fileNameBuilder := f.fileNameBuilder
		if fileNameBuilder == nil {
			fileNameBuilder = DefaultFileNameBuilder
		}
		f.builderMutex.Unlock()

		_ = handle.Mutate(func(report *operation.ExportReport) {
			report.Total = len(rids)
		})

		for _, cardID := range rids {
			err := f.exportLatestCard(handle.Context(), fileNameBuilder, cardID, exportPath, maxExportSize)
			f.exportRequestCache.Push(cardID, handle.ID())
			_ = handle.Mutate(func(report *operation.ExportReport) {
				report.Progress++
				if err == nil {
					report.NoSuccesses++
				} else {
					report.NoFailures++
				}
			})
		}
	}()

	return handle.ID(), nil
}

func (f *Facade) exportLatestCard(ctx context.Context, fileNameBuilder FileNameBuilder, rid resource.RID, exportPath string, maxExportSize int) error {
	f.tracker.LockItem(rid)
	defer f.tracker.UnlockItem(rid)
	catalog := f.vault.Catalog.WithContext(ctx)
	rec, err := catalog.FindRecord(rid)
	if err != nil {
		return err
	}
	dstPath := filepath.Join(exportPath, fileNameBuilder(rec))

	b, err := catalog.GetCardBytes(rid, rec.UpdateTime)
	if err != nil {
		return err
	}

	processor := png.FromBytes(b)
	width, height := processor.ImageSize()
	if width > maxExportSize || height > maxExportSize {
		rawCard, err := processor.Get()
		if err != nil {
			return err
		}
		if err := rawCard.ScaleDown(maxExportSize); err != nil {
			return err
		}
		return rawCard.ToFile(dstPath)
	}

	if err := os.WriteFile(dstPath, b, filex.FilePermission); err != nil {
		return trace.Err().Wrap(err).
			Field(trace.SOURCE, rec.InfoData.Source).
			Field("cardID", rec.ID).
			Msg("Could not copy card for export")
	}

	_ = catalog.UpdateExportData(rid, resource.ExportData{
		ExportTime:      timestamp.Now[timestamp.Nano](),
		ExportedVersion: rec.UpdateTime,
	})
	return nil
}
