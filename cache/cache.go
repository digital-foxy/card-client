package cache

import (
	"slices"
	"sync"

	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/store/resource"
)

type RequestCache struct {
	mu           sync.Mutex
	rids         []resource.RID
	operationIDs []operation.ID
}

func NewRequestCache(initialCapacity int) *RequestCache {
	return &RequestCache{
		rids:         make([]resource.RID, 0, initialCapacity),
		operationIDs: make([]operation.ID, 0, initialCapacity),
	}
}

func (c *RequestCache) HasRequests() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.rids) > 0
}

func (c *RequestCache) Push(rid resource.RID, operationID operation.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rids = append(c.rids, rid)
	c.operationIDs = append(c.operationIDs, operationID)
}

func (c *RequestCache) Flush() ([]resource.RID, []operation.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.rids) == 0 {
		return nil, nil
	}

	flushedCardIDs := slices.Clone(c.rids)
	flushedOperationIDs := slices.Clone(c.operationIDs)

	c.rids = c.rids[:0]
	c.operationIDs = c.operationIDs[:0]

	return flushedCardIDs, flushedOperationIDs
}
