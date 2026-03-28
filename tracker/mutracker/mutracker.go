package mutracker

import (
	"sync"

	"github.com/digital-foxy/card-client/store/resource"
)

// MuTracker is a tracker that uses mutexes to lock items
type MuTracker struct {
	mutex    sync.RWMutex
	trackers map[resource.RID]*resourceMu
}

// New creates a new MuTracker
func New() *MuTracker {
	return &MuTracker{
		trackers: make(map[resource.RID]*resourceMu),
	}
}

// LockItem locks the item with the given RID
func (s *MuTracker) LockItem(rid resource.RID) {
	// Check if the item exists
	s.mutex.RLock()
	state, exists := s.trackers[rid]
	s.mutex.RUnlock()

	// If it exists, lock it immediately
	if exists {
		state.lock()
		return
	}

	// Otherwise, create a new one and lock it
	// Lock the top mutex level for writing
	s.mutex.Lock()
	// Check again, in case another goroutine created it in the meantime
	state, exists = s.trackers[rid]
	if !exists {
		// Create a new mutex
		state = newResourceMu()
		// Add it to the map
		s.trackers[rid] = state
	}
	// Unlock the top mutex level
	s.mutex.Unlock()

	// Lock the item
	state.lock()
}

// UnlockItem unlocks the item with the given RID
func (s *MuTracker) UnlockItem(rid resource.RID) {
	// Lock the top mutex level for reading
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Unlock the item
	if state, exists := s.trackers[rid]; exists {
		state.unlock()
	}
}

// IsItemLocked returns true if the item with the given RID is locked
func (s *MuTracker) IsItemLocked(rid resource.RID) bool {
	// Lock the top mutex level for reading
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if the item exists
	state, exists := s.trackers[rid]
	if !exists {
		return false
	}

	// Return the state
	return state.isLocked()
}

// LockedItems returns a slice of all the items that are currently locked
func (s *MuTracker) LockedItems() []resource.RID {
	// Lock the top mutex level for reading
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Iterate over the map and collect the locked items
	var locked []resource.RID
	for cardID, state := range s.trackers {
		if state.isLocked() {
			locked = append(locked, cardID)
		}
	}
	// Return the slice
	return locked
}
