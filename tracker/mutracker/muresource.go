package mutracker

import (
	"sync"
	"sync/atomic"
)

type resourceMu struct {
	sync.Mutex
	locked uint32
}

func newResourceMu() *resourceMu {
	return &resourceMu{}
}

func (t *resourceMu) lock() {
	t.Lock()
	atomic.StoreUint32(&t.locked, 1)
}

func (t *resourceMu) unlock() {
	if atomic.CompareAndSwapUint32(&t.locked, 1, 0) {
		t.Unlock()
	}
}

func (t *resourceMu) isLocked() bool {
	return atomic.LoadUint32(&t.locked) == 1
}
