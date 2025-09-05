package filter

import (
	"github.com/r3dpixel/card-client/internal/ent/schema"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/source"
)

type PublicField string

const (
	FieldCardName      = PublicField(schema.FieldCardName)
	FieldCharacterName = PublicField(schema.FieldCardCharacterName)
	FieldCreator       = PublicField(schema.FieldCardCreator)
	FieldUpdatedTime   = PublicField(schema.FieldCardUpdateTime)
	FieldCreatedTime   = PublicField(schema.FieldCardCreateTime)
	FieldImportedTime  = PublicField(schema.FieldCardImportTime)
	FieldFavorite      = PublicField(schema.FieldCardFavorite)
)

type FieldControl struct {
	Field       PublicField
	DisplayName string
}

type Controls struct {
	TextFields     []FieldControl
	SortableFields []FieldControl
	BooleanFields  []FieldControl
}

type SortOrder byte

const (
	ASCENDING SortOrder = iota
	DESCENDING
)

type TextMatchMode struct {
	Regex         bool
	WholeWord     bool
	CaseSensitive bool
}

type SearchFilter struct {
	TextFilter     []TextFilter
	SortFilters    []SortFilter
	BooleanFilters []BooleanFilter
	Sources        []source.ID
	Statuses       []scheme.UpdateStatus
}

type TextFilter struct {
	Field     PublicField
	Value     string
	MatchMode TextMatchMode
}

type SortFilter struct {
	Field     PublicField
	Direction SortOrder
}

type BooleanFilter struct {
	Field PublicField
	Value bool
}
