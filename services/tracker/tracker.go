package tracker

import (
	"github.com/r3dpixel/card-client/services/scheme"
)

type Service interface {
	LockItem(cardID scheme.CardID)
	UnlockItem(cardID scheme.CardID)
	IsItemLocked(cardID scheme.CardID) bool
	LockedItems() []scheme.CardID
}
