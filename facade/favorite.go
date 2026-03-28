package facade

import (
	"github.com/digital-foxy/card-client/store/resource"
)

// favoriteService handles favorite operations
type favoriteService struct {
	vault *vaultManager
}

func newFavoriteService(vault *vaultManager) *favoriteService {
	return &favoriteService{
		vault: vault,
	}
}

func (s *favoriteService) ToggleFavorite(rid resource.RID) error {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return err
	}
	defer unlock()

	return vault.Catalog.ToggleFavorite(rid)
}

func (s *favoriteService) UpdateFavorites(favorite bool, rids ...resource.RID) error {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return err
	}
	defer unlock()

	return vault.Catalog.UpdateFavoriteData(favorite, rids...)
}
