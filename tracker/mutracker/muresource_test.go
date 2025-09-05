package mutracker

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracker_InitialState(t *testing.T) {
	tr := newResourceMutex()

	assert.NotNil(t, tr, "newResourceMutex() should not return nil")
	assert.False(t, tr.isLocked(), "A new resourceMu should be in an unlocked state")
}

func TestTracker_LockUnlockCycle(t *testing.T) {
	tr := newResourceMutex()

	tr.lock()
	assert.True(t, tr.isLocked(), "isLocked() should return true after lock()")

	tr.unlock()
	assert.False(t, tr.isLocked(), "isLocked() should return false after unlock()")
}

func TestTracker_MutexBlocks(t *testing.T) {
	tr := newResourceMutex()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tr.lock()
		time.Sleep(100 * time.Millisecond)
		tr.unlock()
	}()

	time.Sleep(20 * time.Millisecond)

	require.True(t, tr.isLocked(), "Tracker should be locked by the first goroutine")

	lockAcquiredBySecondGoRoutine := make(chan struct{})
	go func() {
		tr.lock()
		close(lockAcquiredBySecondGoRoutine)
		tr.unlock()
	}()

	select {
	case <-lockAcquiredBySecondGoRoutine:
		t.Fatal("Second goroutine acquired lock immediately, but it should have been blocked")
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case <-lockAcquiredBySecondGoRoutine:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Second goroutine timed out and failed to acquire the lock after it was released")
	}

	wg.Wait()
}
