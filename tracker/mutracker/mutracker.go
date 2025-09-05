package mutracker

import (
	"sync"

	"github.com/r3dpixel/card-client/serv/scheme"
)

type Service struct {
	mutex    sync.RWMutex
	trackers map[scheme.CardID]*resourceMu
}

func NewService() *Service {
	return &Service{
		trackers: make(map[scheme.CardID]*resourceMu),
	}
}

func (s *Service) LockItem(card scheme.CardID) {
	s.mutex.RLock()
	state, exists := s.trackers[card]
	s.mutex.RUnlock()

	if exists {
		state.lock()
		return
	}

	s.mutex.Lock()
	state, exists = s.trackers[card]
	if !exists {
		state = newResourceMutex()
		s.trackers[card] = state
	}
	s.mutex.Unlock()

	state.lock()
}

func (s *Service) UnlockItem(card scheme.CardID) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if state, exists := s.trackers[card]; exists {
		state.unlock()
	}
}

func (s *Service) IsItemLocked(card scheme.CardID) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	state, exists := s.trackers[card]
	if !exists {
		return false
	}

	return state.isLocked()
}

func (s *Service) LockedItems() []scheme.CardID {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var locked []scheme.CardID
	for cardID, state := range s.trackers {
		if state.isLocked() {
			locked = append(locked, cardID)
		}
	}
	return locked
}
