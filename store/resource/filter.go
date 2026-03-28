package resource

import "github.com/digital-foxy/card-fetcher/source"

// Mode represents the matching mode for filters (ALL or ANY)
type Mode byte

const (
	ALL Mode = iota
	ANY
)

// FieldControls defines the available filter controls for the UI
type FieldControls struct {
	TextFields     []FieldControl
	SortableFields []FieldControl
	BooleanFields  []FieldControl
	Sources        []source.ID
	Statuses       []SyncStatus
	ContentFields  []FieldControl
}

// FieldControl represents a filterable field with its display name
type FieldControl struct {
	Field       Field
	DisplayName string
}

// Filter contains all active filter criteria for querying records
type Filter struct {
	TextFilter     []TextFilter
	BooleanFilters []BooleanFilter
	SortFilters    []SortFilter
	Sources        []source.ID
	Statuses       []SyncStatus
	ContentFilters []ContentFilter
	TagFilter      TagFilter
}

// TextMatchMode defines how text matching should be performed
type TextMatchMode struct {
	Regex         bool
	WholeWord     bool
	CaseSensitive bool
}

// TextFilter represents a text-based filter criterion
type TextFilter struct {
	Field     Field
	Value     string
	MatchMode TextMatchMode
}

// BooleanFilter represents a boolean filter criterion
type BooleanFilter struct {
	Field Field
	Value bool
}

// SortOrder defines the direction of sorting
type SortOrder byte

const (
	ASCENDING SortOrder = iota
	DESCENDING
)

// SortFilter represents a sorting criterion
type SortFilter struct {
	Field     Field
	Direction SortOrder
}

// ContentFilter represents a filter for searching within card content fields
type ContentFilter struct {
	FieldMode Mode
	Fields    []string
	ValueMode Mode
	Values    []string
}

// TagFilter represents a filter for searching by tags
type TagFilter struct {
	Names []string
	Mode  Mode
}
