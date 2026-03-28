package mutracker

import (
	"sync"
	"sync/atomic"
)

// resourceMu is a mutex that tracks whether it is locked or not
type resourceMu struct {
	sync.Mutex
	locked uint32
}

// newResourceMu creates a new resourceMu
func newResourceMu() *resourceMu {
	return &resourceMu{}
}

// lock locks the mutex and sets the locked flag to true
func (t *resourceMu) lock() {
	t.Lock()
	atomic.StoreUint32(&t.locked, 1)
}

// unlock unlocks the mutex and sets the locked flag to false
func (t *resourceMu) unlock() {
	if atomic.CompareAndSwapUint32(&t.locked, 1, 0) {
		t.Unlock()
	}
}

// isLocked returns true if the mutex is locked, false otherwise
func (t *resourceMu) isLocked() bool {
	return atomic.LoadUint32(&t.locked) == 1
}
