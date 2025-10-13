package mutracker

import (
	"sync"

	"github.com/r3dpixel/card-client/store/resource"
)

type MuTracker struct {
	mutex    sync.RWMutex
	trackers map[resource.RID]*resourceMu
}

func New() *MuTracker {
	return &MuTracker{
		trackers: make(map[resource.RID]*resourceMu),
	}
}

func (s *MuTracker) LockItem(rid resource.RID) {
	s.mutex.RLock()
	state, exists := s.trackers[rid]
	s.mutex.RUnlock()

	if exists {
		state.lock()
		return
	}

	s.mutex.Lock()
	state, exists = s.trackers[rid]
	if !exists {
		state = newResourceMu()
		s.trackers[rid] = state
	}
	s.mutex.Unlock()

	state.lock()
}

func (s *MuTracker) UnlockItem(rid resource.RID) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if state, exists := s.trackers[rid]; exists {
		state.unlock()
	}
}

func (s *MuTracker) IsItemLocked(rid resource.RID) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	state, exists := s.trackers[rid]
	if !exists {
		return false
	}

	return state.isLocked()
}

func (s *MuTracker) LockedItems() []resource.RID {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var locked []resource.RID
	for cardID, state := range s.trackers {
		if state.isLocked() {
			locked = append(locked, cardID)
		}
	}
	return locked
}
