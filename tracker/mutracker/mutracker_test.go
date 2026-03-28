package mutracker

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/digital-foxy/card-client/store/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	s := New()
	require.NotNil(t, s)
	assert.NotNil(t, s.trackers)
	assert.Empty(t, s.trackers)
}

func TestService_SingleItemLifecycle(t *testing.T) {
	s := New()
	rid := resource.RID(rand.Int())

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
	testCases := []struct {
		name         string
		setupFunc    func(*MuTracker) resource.RID
		expectError  bool
		validateFunc func(*testing.T, *MuTracker, resource.RID)
	}{
		{
			name: "Unlock non-existent item",
			setupFunc: func(s *MuTracker) resource.RID {
				return resource.RID(rand.Int())
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *MuTracker, rid resource.RID) {
				assert.NotPanics(t, func() {
					s.UnlockItem(rid)
				})
			},
		},
		{
			name: "Unlock already unlocked item",
			setupFunc: func(s *MuTracker) resource.RID {
				rid := resource.RID(rand.Int())
				s.LockItem(rid)
				s.UnlockItem(rid)
				return rid
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *MuTracker, rid resource.RID) {
				assert.NotPanics(t, func() {
					s.UnlockItem(rid)
				})
			},
		},
		{
			name: "IsItemLocked for non-existent item",
			setupFunc: func(s *MuTracker) resource.RID {
				return resource.RID(rand.Int())
			},
			expectError: false,
			validateFunc: func(t *testing.T, s *MuTracker, rid resource.RID) {
				assert.False(t, s.IsItemLocked(rid))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			rid := tc.setupFunc(s)
			tc.validateFunc(t, s, rid)
		})
	}
}

func TestService_LockedItems(t *testing.T) {
	s := New()
	card1, card2, card3 := resource.RID(rand.Int()), resource.RID(rand.Int()), resource.RID(rand.Int())

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
	s := New()
	card1, card2 := resource.RID(rand.Int()), resource.RID(rand.Int())
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
	s := New()
	rid := resource.RID(rand.Int())
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
	s := New()
	rid := resource.RID(rand.Int())

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
	s := New()
	numItems := 5
	numGoroutines := 50
	var rids []resource.RID
	for i := 0; i < numItems; i++ {
		rids = append(rids, resource.RID(rand.Int()))
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
