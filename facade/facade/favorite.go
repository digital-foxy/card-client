package facade

import (
	"context"
)

func (s *Service) ToggleFavorite(cardID scheme.CardID) error {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return err
	} else {
		defer unlock()
	}
	return s.storeService.ToggleFavorite(context.Background(), cardID)
}

func (s *Service) SetFavorites(cardIDs []scheme.CardID, favorite bool) error {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return err
	} else {
		defer unlock()
	}
	return s.storeService.SetFavorites(context.Background(), cardIDs, favorite)
}
