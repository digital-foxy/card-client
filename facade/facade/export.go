package facade

import (
	"context"
	"path/filepath"

	"github.com/r3dpixel/card-client/services/operation"
	"github.com/r3dpixel/card-client/services/preferences"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/trace"
)

func (s *Service) ExportLatestCards(cardIDs ...scheme.CardID) (operation.ID, error) {
	unlock, err := s.beginReadStoreOp()
	if err != nil {
		return operation.EmptyID, err
	}
	handle := s.registry.RegisterExport(s.storeService.VaultName())
	maxExportSize := s.pref.GetInt(preferences.MaxExportSizeKey.ID)

	go func() {
		defer unlock()
		defer handle.Complete()

		exportPath := s.pref.GetString(preferences.ExportPathKey.ID)
		s.builderMutex.Lock()
		fileNameBuilder := s.fileNameBuilder
		if fileNameBuilder == nil {
			fileNameBuilder = DefaultFileNameBuilder
		}
		s.builderMutex.Unlock()

		_ = handle.Mutate(func(report *operation.ExportReport) {
			report.Total = len(cardIDs)
		})

		for _, cardID := range cardIDs {
			err := s.exportLatestCard(handle.Context(), fileNameBuilder, cardID, exportPath, maxExportSize)
			s.exportRequestCache.Push(cardID, handle.ID())
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

func (s *Service) exportLatestCard(ctx context.Context, fileNameBuilder FileNameBuilder, cardID scheme.CardID, exportPath string, maxExportSize int) error {
	s.trackerService.LockItem(cardID)
	defer s.trackerService.UnlockItem(cardID)
	header, err := s.storeService.FindMiscHeader(ctx, cardID)
	if err != nil {
		return err
	}
	dstPath := filepath.Join(exportPath, fileNameBuilder(header))

	srcPath := s.storeService.GetPngPath(cardID, header.UpdateTime)

	processor := png.FromFile(srcPath)
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

	if copyErr := filex.CopyFile(srcPath, dstPath); copyErr != nil {
		return trace.Err().Wrap(copyErr).
			Field(trace.SOURCE, header.Source).
			Field("cardID", header.CardID).
			Msg("Could not copy card for export")
	}

	_ = s.storeService.UpdateToLatestExport(ctx, cardID, header.UpdateTime)
	return nil
}
