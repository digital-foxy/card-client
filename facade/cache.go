package facade

import (
	"github.com/r3dpixel/card-client/cache"
	"github.com/r3dpixel/card-client/store/resource"
)

func (f *Facade) FlushUpdateCache() (resource.Box[OperationPayload], error) {
	return f.flushCache(f.updateRequestCache)
}

func (f *Facade) FlushExportCache() (resource.Box[OperationPayload], error) {
	return f.flushCache(f.updateRequestCache)
}

func (f *Facade) HasUpdatePayloadRequests() bool {
	return f.updateRequestCache.HasRequests()
}

func (f *Facade) HasExportPayloadRequests() bool {
	return f.exportRequestCache.HasRequests()
}

func (f *Facade) flushCache(cache *cache.RequestCache) (resource.Box[OperationPayload], error) {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return resource.Box[OperationPayload]{}, err
	} else {
		defer unlock()
	}
	rids, operationIDs := cache.Flush()

	box, err := f.vault.Catalog.FindRecords(rids...)
	if err != nil {
		return resource.Box[OperationPayload]{}, err
	}

	headerMap := make(map[resource.RID]*resource.Record)
	for index := range box.Items {
		headerMap[box.Items[index].ID] = &box.Items[index]
	}

	var payloads []OperationPayload
	for index, cardID := range rids {
		rec, ok := headerMap[cardID]
		if !ok {
			continue
		}
		payloads = append(payloads, OperationPayload{
			OperationID: operationIDs[index],
			Record:      *rec,
		})
	}
	return resource.Box[OperationPayload]{
		Items:     payloads,
		Timestamp: box.Timestamp,
	}, nil
}
