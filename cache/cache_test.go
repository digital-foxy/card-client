package cache

import (
	"fmt"
	"sync"
	"testing"

	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestCache(t *testing.T) {
	cache := NewRequestCache(10)
	require.NotNil(t, cache)
	assert.False(t, cache.HasRequests())
}

func TestRequestCache_HasRequests_Empty(t *testing.T) {
	cache := NewRequestCache(10)
	assert.False(t, cache.HasRequests())
}

func TestRequestCache_HasRequests_WithData(t *testing.T) {
	cache := NewRequestCache(10)
	cache.Push(1, "op-100")
	assert.True(t, cache.HasRequests())
}

func TestRequestCache_Push_Single(t *testing.T) {
	cache := NewRequestCache(10)

	cache.Push(1, "op-100")

	assert.True(t, cache.HasRequests())
	rids, opIDs := cache.Flush()
	assert.Equal(t, []resource.RID{1}, rids)
	assert.Equal(t, []operation.ID{"op-100"}, opIDs)
}

func TestRequestCache_Push_Multiple(t *testing.T) {
	cache := NewRequestCache(10)

	cache.Push(1, "op-100")
	cache.Push(2, "op-200")
	cache.Push(3, "op-300")

	assert.True(t, cache.HasRequests())
	rids, opIDs := cache.Flush()
	assert.Equal(t, []resource.RID{1, 2, 3}, rids)
	assert.Equal(t, []operation.ID{"op-100", "op-200", "op-300"}, opIDs)
}

func TestRequestCache_Flush_Empty(t *testing.T) {
	cache := NewRequestCache(10)

	rids, opIDs := cache.Flush()

	assert.Nil(t, rids)
	assert.Nil(t, opIDs)
}

func TestRequestCache_Flush_ClearsCache(t *testing.T) {
	cache := NewRequestCache(10)

	cache.Push(1, "op-100")
	cache.Push(2, "op-200")

	// First flush
	rids1, opIDs1 := cache.Flush()
	assert.Len(t, rids1, 2)
	assert.Len(t, opIDs1, 2)

	// Cache should be empty now
	assert.False(t, cache.HasRequests())

	// Second flush should return nil
	rids2, opIDs2 := cache.Flush()
	assert.Nil(t, rids2)
	assert.Nil(t, opIDs2)
}

func TestRequestCache_Flush_ReturnsClones(t *testing.T) {
	cache := NewRequestCache(10)

	cache.Push(1, "op-100")
	cache.Push(2, "op-200")

	rids, opIDs := cache.Flush()

	// Modify the returned slices
	rids[0] = 999
	opIDs[0] = "op-999"

	// Push new data and flush again
	cache.Push(3, "op-300")
	rids2, opIDs2 := cache.Flush()

	// New flush should not be affected by modifications to previous flush
	assert.Equal(t, []resource.RID{3}, rids2)
	assert.Equal(t, []operation.ID{"op-300"}, opIDs2)
}

func TestRequestCache_MultipleFlushCycles(t *testing.T) {
	cache := NewRequestCache(10)

	// First cycle
	cache.Push(1, "op-100")
	cache.Push(2, "op-200")
	rids1, opIDs1 := cache.Flush()
	assert.Equal(t, []resource.RID{1, 2}, rids1)
	assert.Equal(t, []operation.ID{"op-100", "op-200"}, opIDs1)

	// Second cycle
	cache.Push(3, "op-300")
	cache.Push(4, "op-400")
	rids2, opIDs2 := cache.Flush()
	assert.Equal(t, []resource.RID{3, 4}, rids2)
	assert.Equal(t, []operation.ID{"op-300", "op-400"}, opIDs2)

	// Third cycle
	cache.Push(5, "op-500")
	rids3, opIDs3 := cache.Flush()
	assert.Equal(t, []resource.RID{5}, rids3)
	assert.Equal(t, []operation.ID{"op-500"}, opIDs3)
}

func TestRequestCache_ExceedsInitialCapacity(t *testing.T) {
	cache := NewRequestCache(2)

	// Push more items than initial capacity
	for i := 1; i <= 10; i++ {
		cache.Push(resource.RID(i), operation.ID(fmt.Sprintf("op-%d", i*100)))
	}

	rids, opIDs := cache.Flush()
	assert.Len(t, rids, 10)
	assert.Len(t, opIDs, 10)
}

func TestRequestCache_Concurrency(t *testing.T) {
	cache := NewRequestCache(100)
	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent pushes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			cache.Push(resource.RID(n), operation.ID(fmt.Sprintf("op-%d", n)))
		}(i)
	}

	wg.Wait()

	rids, opIDs := cache.Flush()
	assert.Len(t, rids, numGoroutines)
	assert.Len(t, opIDs, numGoroutines)
}

func TestRequestCache_ConcurrentPushAndFlush(t *testing.T) {
	cache := NewRequestCache(100)
	var pushWg sync.WaitGroup
	var flushWg sync.WaitGroup
	totalPushed := 0
	totalFlushed := 0
	var flushMutex sync.Mutex

	// Concurrent pushes
	pushWg.Add(50)
	for i := 0; i < 50; i++ {
		go func(n int) {
			defer pushWg.Done()
			cache.Push(resource.RID(n), operation.ID(fmt.Sprintf("op-%d", n)))
		}(i)
	}
	totalPushed += 50

	// Wait for all pushes to complete before flushing
	pushWg.Wait()

	// Concurrent flushes
	flushWg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer flushWg.Done()
			rids, opIDs := cache.Flush()
			flushMutex.Lock()
			totalFlushed += len(rids)
			flushMutex.Unlock()
			assert.Equal(t, len(rids), len(opIDs))
		}()
	}

	flushWg.Wait()

	// All items should have been flushed
	assert.Equal(t, totalPushed, totalFlushed)
	assert.False(t, cache.HasRequests())
}

func TestRequestCache_ConcurrentHasRequests(t *testing.T) {
	cache := NewRequestCache(100)
	var wg sync.WaitGroup

	// Concurrent reads and writes
	wg.Add(100)
	for i := 0; i < 50; i++ {
		go func(n int) {
			defer wg.Done()
			cache.Push(resource.RID(n), operation.ID(fmt.Sprintf("op-%d", n)))
		}(i)
	}

	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			_ = cache.HasRequests()
		}()
	}

	wg.Wait()

	// Should have some requests
	assert.True(t, cache.HasRequests())
}
