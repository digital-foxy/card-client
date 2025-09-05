package facade

import (
	"context"

	"github.com/r3dpixel/card-client/services/filter"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/toolkit/timestamp"
)

func (s *Service) Count() (int, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return 0, err
	} else {
		defer unlock()
	}
	return s.storeService.Count(context.Background()), nil
}

func (s *Service) FindIDs(filter filter.SearchFilter) ([]scheme.CardID, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return []scheme.CardID{}, err
	} else {
		defer unlock()
	}
	headers := s.storeService.FindPagedIDs(context.Background(), filter, 0, -1)
	return headers, nil
}
func (s *Service) FindPagedIDs(filter filter.SearchFilter, offset int, limit int) ([]scheme.CardID, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return []scheme.CardID{}, err
	} else {
		defer unlock()
	}
	headers := s.storeService.FindPagedIDs(context.Background(), filter, offset, limit)
	return headers, nil
}

func (s *Service) FindCards(cardIDs ...scheme.CardID) ([]scheme.CardView, timestamp.Nano, error) {
	if unlock, err := s.beginReadStoreOp(); err != nil {
		return []scheme.CardView{}, 0, err
	} else {
		defer unlock()
	}
	headers, readAt := s.storeService.FindCards(context.Background(), cardIDs)
	views := make([]scheme.CardView, len(headers))
	for index, header := range headers {
		views[index] = scheme.CardView{
			CardHeader: header,
			Thumbnail:  s.storeService.GetThumbnailPath(header.CardID.String()),
		}
	}
	return views, readAt, nil

}
