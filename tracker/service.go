package tracker

import (
	"github.com/r3dpixel/card-client/store/resource"
)

type Service interface {
	Lock(rid resource.RID) error
	Unlock(rid resource.RID)
	IsLocked(rid resource.RID) bool
	Locked() []resource.RID
}
