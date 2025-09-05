package tracker

import (
	"github.com/r3dpixel/card-client/serv/scheme"
)

type Service interface {
	Lock(cardID scheme.CardID)
	Unlock(cardID scheme.CardID)
	IsLocked(cardID scheme.CardID) bool
	Locked() []scheme.CardID
}
