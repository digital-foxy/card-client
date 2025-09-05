package facade

import (
	"context"

	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/toolkit/timestamp"
)

func (s *Service) FlushUpdatePayloads() ([]scheme.UpdatePayload, timestamp.Nano, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return []scheme.UpdatePayload{}, 0, err
	} else {
		defer unlock()
	}
	cardIDs, operationIDs := s.updateRequestCache.Flush()

	headers, readAt := s.storeService.FindCards(context.Background(), cardIDs)
	headerMap := make(map[scheme.CardID]*scheme.CardHeader)
	for index, _ := range headers {
		headerMap[headers[index].CardID] = &headers[index]
	}

	var payloads []scheme.UpdatePayload
	for index, cardID := range cardIDs {
		header, ok := headerMap[cardID]
		if !ok || header == nil {
			continue
		}
		payloads = append(payloads, scheme.UpdatePayload{
			CardID:       cardID,
			OperationID:  operationIDs[index],
			DataHeader:   header.DataHeader,
			UpdateHeader: header.UpdateHeader,
		})
	}
	return payloads, readAt, nil
}

func (s *Service) FlushExportPayloads() ([]scheme.ExportPayload, timestamp.Nano, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return []scheme.ExportPayload{}, 0, err
	} else {
		defer unlock()
	}

	cardIDs, operationIDs := s.exportRequestCache.Flush()

	headers, readAt := s.storeService.FindIdExportHeaders(context.Background(), cardIDs)
	headerMap := make(map[scheme.CardID]scheme.IdExportHeader)
	for index, _ := range headers {
		headerMap[headers[index].CardID] = headers[index]
	}

	var payloads []scheme.ExportPayload
	for index, cardID := range cardIDs {
		header, ok := headerMap[cardID]
		if !ok {
			continue
		}

		payloads = append(payloads, scheme.ExportPayload{
			OperationID:  operationIDs[index],
			CardID:       cardID,
			ExportHeader: header.ExportHeader,
		})
	}

	return payloads, readAt, nil
}

func (s *Service) HasUpdatePayloadRequests() bool {
	return s.updateRequestCache.HasRequests()
}

func (s *Service) HasExportPayloadRequests() bool {
	return s.exportRequestCache.HasRequests()
}
