package tracker

import (
	"github.com/r3dpixel/card-client/store/resource"
)

type Service interface {
	LockItem(rid resource.RID)
	UnlockItem(rid resource.RID)
	IsItemLocked(rid resource.RID) bool
	LockedItems() []resource.RID
}
