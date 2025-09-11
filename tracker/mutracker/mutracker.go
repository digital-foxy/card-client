package mutracker

import (
	"sync"

	"github.com/r3dpixel/card-client/store/resource"
)

type Service struct {
	mutex    sync.RWMutex
	trackers map[resource.RID]*resourceMu
}

func NewService() *Service {
	return &Service{
		trackers: make(map[resource.RID]*resourceMu),
	}
}

func (s *Service) LockItem(rid resource.RID) {
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
		state = newResourceMutex()
		s.trackers[rid] = state
	}
	s.mutex.Unlock()

	state.lock()
}

func (s *Service) UnlockItem(rid resource.RID) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if state, exists := s.trackers[rid]; exists {
		state.unlock()
	}
}

func (s *Service) IsItemLocked(rid resource.RID) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	state, exists := s.trackers[rid]
	if !exists {
		return false
	}

	return state.isLocked()
}

func (s *Service) LockedItems() []resource.RID {
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
