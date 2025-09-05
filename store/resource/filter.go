package resource

import "github.com/r3dpixel/card-fetcher/source"

type FieldControls struct {
	TextFields     []FieldControl
	SortableFields []FieldControl
	BooleanFields  []FieldControl
}

type FieldControl struct {
	Field       Field
	DisplayName string
}

type Filter struct {
	TextFilter     []TextFilter
	BooleanFilters []BooleanFilter
	SortFilters    []SortFilter
	Sources        []source.ID
	Statuses       []SyncStatus
}

type TextMatchMode struct {
	Regex         bool
	WholeWord     bool
	CaseSensitive bool
}

type TextFilter struct {
	Field     Field
	Value     string
	MatchMode TextMatchMode
}

type BooleanFilter struct {
	Field Field
	Value bool
}

type SortOrder byte

const (
	ASCENDING SortOrder = iota
	DESCENDING
)

type SortFilter struct {
	Field     Field
	Direction SortOrder
}

func GetFilterControls() FieldControls {
	return FieldControls{
		TextFields: []FieldControl{
			{FieldCardName, "Card Name"},
			{FieldCardCreator, "Creator"},
			{FieldCardCharacterName, "Name"},
		},
		SortableFields: []FieldControl{
			{FieldCardUpdateTime, "UpdateInfoSyncData Time"},
			{FieldCardCreateTime, "Create Time"},
			{FieldCardImportTime, "Import Time"},
		},
		BooleanFields: []FieldControl{
			{FieldCardFavorite, "Favorite"},
		},
	}
}
