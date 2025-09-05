package cache

import (
	"slices"
	"sync"

	"github.com/r3dpixel/card-client/services/operation"
	"github.com/r3dpixel/card-client/services/scheme"
)

type RequestCache struct {
	mu           sync.Mutex
	cardIDs      []scheme.CardID
	operationIDs []operation.ID
}

func NewRequestCache(initialCapacity int) *RequestCache {
	return &RequestCache{
		cardIDs:      make([]scheme.CardID, 0, initialCapacity),
		operationIDs: make([]operation.ID, 0, initialCapacity),
	}
}

func (c *RequestCache) HasRequests() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cardIDs) > 0
}

func (c *RequestCache) Push(cardID scheme.CardID, operationID operation.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cardIDs = append(c.cardIDs, cardID)
	c.operationIDs = append(c.operationIDs, operationID)
}

func (c *RequestCache) Flush() ([]scheme.CardID, []operation.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cardIDs) == 0 {
		return nil, nil
	}

	flushedCardIDs := slices.Clone(c.cardIDs)
	flushedOperationIDs := slices.Clone(c.operationIDs)

	c.cardIDs = c.cardIDs[:0]
	c.operationIDs = c.operationIDs[:0]

	return flushedCardIDs, flushedOperationIDs
}
