package filter

import (
	"github.com/r3dpixel/card-client/services/filter"
)

type Service struct {
}

func (s *Service) GetFilterControls() filter.Controls {
	return filter.Controls{
		TextFields: []filter.FieldControl{
			{filter.FieldCardName, "Card Name"},
			{filter.FieldCreator, "Creator"},
			{filter.FieldCharacterName, "Name"},
		},
		SortableFields: []filter.FieldControl{
			{filter.FieldUpdatedTime, "Update Time"},
			{filter.FieldCreatedTime, "Create Time"},
			{filter.FieldImportedTime, "Import Time"},
		},
		BooleanFields: []filter.FieldControl{
			{filter.FieldFavorite, "Favorite"},
		},
	}
}
