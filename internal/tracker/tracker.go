package tracker

import (
	"sync"
	"sync/atomic"
)

type tracker struct {
	mutex  sync.Mutex
	locked uint32
}

func newTracker() *tracker {
	return &tracker{}
}

func (t *tracker) lock() {
	t.mutex.Lock()
	atomic.StoreUint32(&t.locked, 1)
}

func (t *tracker) unlock() {
	if atomic.CompareAndSwapUint32(&t.locked, 1, 0) {
		t.mutex.Unlock()
	}
}

func (t *tracker) isLocked() bool {
	return atomic.LoadUint32(&t.locked) == 1
}
