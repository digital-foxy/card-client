package mutracker

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	s := NewService()
	require.NotNil(t, s)
	assert.NotNil(t, s.trackers)
	assert.Empty(t, s.trackers)
}

func TestService_SingleItemLifecycle(t *testing.T) {
	s := NewService()
	rid := resource.RID(uuid.NewString())

	assert.False(t, s.IsItemLocked(rid))
	assert.Empty(t, s.LockedItems())

	s.LockItem(rid)
	assert.True(t, s.IsItemLocked(rid))
	assert.Equal(t, []resource.RID{rid}, s.LockedItems())

	s.UnlockItem(rid)
	assert.False(t, s.IsItemLocked(rid))
	assert.Empty(t, s.LockedItems())
}

func TestService_EdgeCases(t *testing.T) {
	t.Run("Unlock non-existent item", func(t *testing.T) {
		s := NewService()
		rid := resource.RID(uuid.NewString())
		assert.NotPanics(t, func() {
			s.UnlockItem(rid)
		})
	})

	t.Run("Unlock already unlocked item", func(t *testing.T) {
		s := NewService()
		rid := resource.RID(uuid.NewString())
		s.LockItem(rid)
		s.UnlockItem(rid)
		assert.NotPanics(t, func() {
			s.UnlockItem(rid)
		})
	})

	t.Run("IsItemLocked for non-existent item", func(t *testing.T) {
		s := NewService()
		rid := resource.RID(uuid.NewString())
		assert.False(t, s.IsItemLocked(rid))
	})
}

func TestService_LockedItems(t *testing.T) {
	s := NewService()
	card1, card2, card3 := resource.RID(uuid.NewString()), resource.RID(uuid.NewString()), resource.RID(uuid.NewString())

	assert.Empty(t, s.LockedItems())

	s.LockItem(card1)
	s.LockItem(card2)
	s.LockItem(card3)

	locked := s.LockedItems()
	expected := []resource.RID{card1, card2, card3}
	assert.ElementsMatch(t, expected, locked)

	s.UnlockItem(card2)
	lockedAfterUnlock := s.LockedItems()
	expectedAfterUnlock := []resource.RID{card1, card3}
	assert.ElementsMatch(t, expectedAfterUnlock, lockedAfterUnlock)
}

func TestService_Concurrency_LockDifferentItems(t *testing.T) {
	s := NewService()
	card1, card2 := resource.RID(uuid.NewString()), resource.RID(uuid.NewString())
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.LockItem(card1)
	}()
	go func() {
		defer wg.Done()
		s.LockItem(card2)
	}()
	wg.Wait()

	assert.True(t, s.IsItemLocked(card1))
	assert.True(t, s.IsItemLocked(card2))
	assert.Len(t, s.LockedItems(), 2)
}

func TestService_Concurrency_RaceToCreateSameItem(t *testing.T) {
	s := NewService()
	rid := resource.RID(uuid.NewString())
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func() {
			s.LockItem(rid)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	s.mutex.RLock()
	assert.Len(t, s.trackers, 1, "Only one tracker should have been created for the same item")
	s.mutex.RUnlock()

	assert.True(t, s.IsItemLocked(rid), "The item should be locked after the race")
}

func TestService_Concurrency_BlockOnSameItemLock(t *testing.T) {
	s := NewService()
	rid := resource.RID(uuid.NewString())

	s.LockItem(rid)
	require.True(t, s.IsItemLocked(rid))

	lockAcquiredBySecondGoroutine := make(chan struct{})
	go func() {
		s.LockItem(rid)
		close(lockAcquiredBySecondGoroutine)
	}()

	select {
	case <-lockAcquiredBySecondGoroutine:
		t.Fatal("Second goroutine acquired lock immediately but should have blocked")
	case <-time.After(50 * time.Millisecond):
	}

	require.True(t, s.IsItemLocked(rid))

	s.UnlockItem(rid)

	select {
	case <-lockAcquiredBySecondGoroutine:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Second goroutine failed to acquire the lock after it was released")
	}
}

func TestService_Concurrency_HeavyContention(t *testing.T) {
	s := NewService()
	numItems := 5
	numGoroutines := 50
	var rids []resource.RID
	for i := 0; i < numItems; i++ {
		rids = append(rids, resource.RID(uuid.NewString()))
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			rid := rids[n%numItems]
			s.LockItem(rid)
			time.Sleep(time.Duration(n%5) * time.Millisecond)
			s.UnlockItem(rid)
		}(i)
	}
	wg.Wait()

	assert.Empty(t, s.LockedItems())
	assert.Len(t, s.trackers, numItems)
}
