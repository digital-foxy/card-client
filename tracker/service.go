package tracker

import (
	"github.com/digital-foxy/card-client/store/resource"
)

// Service Tracker service (handle locking per item)
type Service interface {
	// LockItem locks the item with the given RID
	LockItem(rid resource.RID)

	// UnlockItem unlocks the item with the given RID
	UnlockItem(rid resource.RID)

	// IsItemLocked returns true if the item with the given RID is locked
	IsItemLocked(rid resource.RID) bool

	// LockedItems returns a slice of all the items that are currently locked
	LockedItems() []resource.RID
}
