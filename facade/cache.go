package facade

import (
	"slices"
	"sync"

	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/timestamp"
)

// cacheManager buffers operation results for batch retrieval
type cacheManager struct {
	vault *vaultManager
	rids  []resource.RID
	opIDs []operation.ID
	mu    sync.RWMutex
}

func newCacheManager(vault *vaultManager) *cacheManager {
	return &cacheManager{
		vault: vault,
		rids:  make([]resource.RID, 0, 128),
		opIDs: make([]operation.ID, 0, 128),
	}
}

func (m *cacheManager) Push(rid resource.RID, operationID operation.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rids = append(m.rids, rid)
	m.opIDs = append(m.opIDs, operationID)
}

func (m *cacheManager) HasRequests() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rids) > 0
}

func (m *cacheManager) Flush() (resource.Box[OperationPayload], error) {
	if !m.HasRequests() {
		return resource.Box[OperationPayload]{
			Timestamp: timestamp.NowNano(),
		}, nil
	}

	m.mu.Lock()
	rids := slices.Clone(m.rids)
	operationIDs := slices.Clone(m.opIDs)

	m.rids = m.rids[:0]
	m.opIDs = m.opIDs[:0]
	m.mu.Unlock()

	vault, unlock, err := m.vault.beginReadOp()
	if err != nil {
		// Can't access vault, return all as missing
		return m.buildMissingPayloads(rids, operationIDs, timestamp.NowNano()), nil
	}
	defer unlock()

	box, err := vault.Catalog.FindRecords(rids...)
	if err != nil {
		// DB read failed, return all as missing
		return m.buildMissingPayloads(rids, operationIDs, box.Timestamp), nil
	}

	recordMap := make(map[resource.RID]*resource.Record, len(box.Items))
	for index := range box.Items {
		recordMap[box.Items[index].ID] = &box.Items[index]
	}

	var payloads []OperationPayload
	for index, rid := range rids {
		rec, ok := recordMap[rid]
		if !ok {
			// Record not found, mark as missing
			payloads = append(payloads, OperationPayload{
				OperationID: operationIDs[index],
				Record: resource.Record{
					ID: rid,
					SyncData: resource.SyncData{
						SyncTime:   box.Timestamp,
						SyncStatus: resource.SyncMissing,
					}},
			})
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

func (m *cacheManager) buildMissingPayloads(rids []resource.RID, operationIDs []operation.ID, syncTime timestamp.Nano) resource.Box[OperationPayload] {
	payloads := make([]OperationPayload, len(rids))
	for index, rid := range rids {
		payloads[index] = OperationPayload{
			OperationID: operationIDs[index],
			Record: resource.Record{
				ID: rid,
				SyncData: resource.SyncData{
					SyncTime:   syncTime,
					SyncStatus: resource.SyncMissing,
				}},
		}
	}
	return resource.Box[OperationPayload]{
		Items:     payloads,
		Timestamp: syncTime,
	}
}
