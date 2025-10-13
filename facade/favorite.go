package facade

import (
	"github.com/r3dpixel/card-client/store/resource"
)

func (f *Facade) ToggleFavorite(rid resource.RID) error {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return err
	} else {
		defer unlock()
	}
	return f.vault.Catalog.ToggleFavorite(rid)
}

func (f *Facade) UpdateFavorites(favorite bool, rids ...resource.RID) error {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return err
	} else {
		defer unlock()
	}
	return f.vault.Catalog.UpdateFavoriteData(favorite, rids...)
}
