package tracker

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/r3dpixel/card-client/services/scheme"
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
	cardID := scheme.CardID(uuid.NewString())

	assert.False(t, s.IsItemLocked(cardID))
	assert.Empty(t, s.LockedItems())

	s.LockItem(cardID)
	assert.True(t, s.IsItemLocked(cardID))
	assert.Equal(t, []scheme.CardID{cardID}, s.LockedItems())

	s.UnlockItem(cardID)
	assert.False(t, s.IsItemLocked(cardID))
	assert.Empty(t, s.LockedItems())
}

func TestService_EdgeCases(t *testing.T) {
	t.Run("Unlock non-existent item", func(t *testing.T) {
		s := NewService()
		cardID := scheme.CardID(uuid.NewString())
		assert.NotPanics(t, func() {
			s.UnlockItem(cardID)
		})
	})

	t.Run("Unlock already unlocked item", func(t *testing.T) {
		s := NewService()
		cardID := scheme.CardID(uuid.NewString())
		s.LockItem(cardID)
		s.UnlockItem(cardID)
		assert.NotPanics(t, func() {
			s.UnlockItem(cardID)
		})
	})

	t.Run("IsItemLocked for non-existent item", func(t *testing.T) {
		s := NewService()
		cardID := scheme.CardID(uuid.NewString())
		assert.False(t, s.IsItemLocked(cardID))
	})
}

func TestService_LockedItems(t *testing.T) {
	s := NewService()
	card1, card2, card3 := scheme.CardID(uuid.NewString()), scheme.CardID(uuid.NewString()), scheme.CardID(uuid.NewString())

	assert.Empty(t, s.LockedItems())

	s.LockItem(card1)
	s.LockItem(card2)
	s.LockItem(card3)

	locked := s.LockedItems()
	expected := []scheme.CardID{card1, card2, card3}
	assert.ElementsMatch(t, expected, locked)

	s.UnlockItem(card2)
	lockedAfterUnlock := s.LockedItems()
	expectedAfterUnlock := []scheme.CardID{card1, card3}
	assert.ElementsMatch(t, expectedAfterUnlock, lockedAfterUnlock)
}

func TestService_Concurrency_LockDifferentItems(t *testing.T) {
	s := NewService()
	card1, card2 := scheme.CardID(uuid.NewString()), scheme.CardID(uuid.NewString())
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
	cardID := scheme.CardID(uuid.NewString())
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func() {
			s.LockItem(cardID)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	s.mutex.RLock()
	assert.Len(t, s.trackers, 1, "Only one tracker should have been created for the same item")
	s.mutex.RUnlock()

	assert.True(t, s.IsItemLocked(cardID), "The item should be locked after the race")
}

func TestService_Concurrency_BlockOnSameItemLock(t *testing.T) {
	s := NewService()
	cardID := scheme.CardID(uuid.NewString())

	s.LockItem(cardID)
	require.True(t, s.IsItemLocked(cardID))

	lockAcquiredBySecondGoroutine := make(chan struct{})
	go func() {
		s.LockItem(cardID)
		close(lockAcquiredBySecondGoroutine)
	}()

	select {
	case <-lockAcquiredBySecondGoroutine:
		t.Fatal("Second goroutine acquired lock immediately but should have blocked")
	case <-time.After(50 * time.Millisecond):
	}

	require.True(t, s.IsItemLocked(cardID))

	s.UnlockItem(cardID)

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
	var cardIDs []scheme.CardID
	for i := 0; i < numItems; i++ {
		cardIDs = append(cardIDs, scheme.CardID(uuid.NewString()))
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			cardID := cardIDs[n%numItems]
			s.LockItem(cardID)
			time.Sleep(time.Duration(n%5) * time.Millisecond)
			s.UnlockItem(cardID)
		}(i)
	}
	wg.Wait()

	assert.Empty(t, s.LockedItems())
	assert.Len(t, s.trackers, numItems)
}
