package resource

import (
	"slices"
	"strings"

	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/property"
	"github.com/digital-foxy/toolkit/slicesx"
	"github.com/digital-foxy/toolkit/stringsx"
	"github.com/digital-foxy/toolkit/timestamp"
)

// AnonymousCreator is the nickname used for anonymous records
var anonymousIdentifier = strings.ToLower(character.AnonymousCreator)

// Box is a generic container that holds a slice of items with a timestamp
type Box[T any] struct {
	Items     []T
	Timestamp timestamp.Nano
}

// Record is the main resource record containing all data for a character card
type Record struct {
	ID RID
	ImportData
	InfoData
	Creator
	SyncData
	ExportData
	AuxData
}

// Integrity returns true if all required fields of the record are valid
func (r *Record) Integrity() bool {
	return stringsx.IsNotBlank(string(r.InfoData.Source)) &&
		stringsx.IsNotBlank(r.NormalizedURL) &&
		stringsx.IsNotBlank(r.DirectURL) &&
		stringsx.IsNotBlank(r.InfoData.PlatformID) &&
		stringsx.IsNotBlank(r.CharacterID) &&
		stringsx.IsNotBlank(r.Name) &&
		stringsx.IsNotBlank(r.Title) &&
		r.CreateTime > 0 &&
		r.UpdateTime > 0 &&
		r.UpdateTime >= r.CreateTime &&
		stringsx.IsNotBlank(r.Nickname) &&
		stringsx.IsNotBlank(r.Username) &&
		(strings.ToLower(r.Creator.Nickname) == anonymousIdentifier || stringsx.IsNotBlank(r.Creator.PlatformID))
}

// FixIntegrity attempts to repair inconsistencies between the record and sheet
func (r *Record) FixIntegrity(sheet *character.Sheet) RecordIntegrity {
	// Check if the record is broken
	if broken := r.isBroken(sheet); broken {
		return BROKEN
	}

	// Sync fields between record and sheet
	updated := r.syncSheetFields(sheet) || r.syncRecordFields(sheet)

	// Return record integrity
	switch {
	// If sheet integrity is still broken, the sync failed
	case !sheet.Integrity():
		return BROKEN
	// If there were updates, the record is fixed
	case updated:
		return FIXED
	// If no updates were made, the record is OK
	default:
		return OK
	}
}

// isBroken returns true if the record is broken
func (r *Record) isBroken(sheet *character.Sheet) bool {
	switch {
	// If the record is missing, but a sheet exists, or vice versa, the record is broken
	case (r == nil) != (sheet == nil):
		return true
	// If the record is nil, the record is broken (checks for nil sheet fields were already performed in the above case)
	case r == nil:
		return false
	// Check record integrity
	case !r.Integrity():
		return true
	// The sheet must have the tagline as the prefix, if it doesn't, the record is broken
	case !strings.HasPrefix(string(sheet.CreatorNotes), r.Tagline):
		return true
	}
	// Return false if the record is OK
	return false
}

// syncSheetFields updates the sheet fields based on the record data
func (r *Record) syncSheetFields(sheet *character.Sheet) bool {
	updated := false

	// Sync source ID
	if string(r.InfoData.Source) != string(sheet.SourceID) {
		sheet.SourceID = property.String(r.InfoData.Source)
		updated = true
	}
	// Sync character ID
	if r.CharacterID != string(sheet.CharacterID) {
		sheet.CharacterID = property.String(r.CharacterID)
		updated = true
	}
	// Sync character platform ID
	if r.InfoData.PlatformID != string(sheet.PlatformID) {
		sheet.PlatformID = property.String(r.InfoData.PlatformID)
		updated = true
	}
	// Sync normalized URL
	if r.DirectURL != string(sheet.DirectLink) {
		sheet.DirectLink = property.String(r.DirectURL)
		updated = true
	}
	// Sync sheet title
	if r.Title != string(sheet.Title) {
		sheet.Title = property.String(r.Title)
		updated = true
	}
	// Sync sheet name
	if r.Name != string(sheet.Name) {
		sheet.Name = property.String(r.Name)
		updated = true
	}
	// Sync creator nickname
	if r.Creator.Nickname != string(sheet.Creator) {
		sheet.Creator = property.String(r.Creator.Nickname)
		updated = true
	}
	// Sync sheet creation date (creation date is timestamp in seconds)
	createDateSeconds := timestamp.ConvertToSeconds(r.CreateTime)
	if createDateSeconds != sheet.CreationDate {
		sheet.CreationDate = createDateSeconds
		updated = true
	}
	// Sync sheet modification date (modification date is timestamp in seconds)
	modificationDateSeconds := timestamp.ConvertToSeconds(r.LatestUpdateTime())
	if modificationDateSeconds != sheet.ModificationDate {
		sheet.ModificationDate = modificationDateSeconds
		updated = true
	}
	// Sync sheet tags
	recordTags := TagNames(r.Tags)
	if !slices.Equal(recordTags, sheet.Tags) {
		sheet.Tags = slices.Clone(recordTags)
		updated = true
	}
	// Return updated flag
	return updated
}

// syncRecordFields updates the record fields based on the sheet data
func (r *Record) syncRecordFields(sheet *character.Sheet) bool {
	updated := false

	// Update greetings count
	if len(sheet.AlternateGreetings) != r.GreetingsCount {
		r.GreetingsCount = len(sheet.AlternateGreetings)
		updated = true
	}

	// Update book update time
	// If a book is not present in the sheet, the record's book update must be 0 (if it's different from 0, update)
	if sheet.CharacterBook == nil && r.BookUpdateTime != 0 {
		r.BookUpdateTime = 0
		updated = true
	}
	// If a book is present in the sheet, and the record's book update time is 0, update it to latest update time
	if sheet.CharacterBook != nil && r.BookUpdateTime == 0 {
		r.BookUpdateTime = r.LatestUpdateTime()
		updated = true
	}

	// Return updated flag
	return updated
}

// ToMetadata converts the record to a metadata model
func (r *Record) ToMetadata() *models.Metadata {
	return &models.Metadata{
		Source: r.InfoData.Source,
		CardInfo: models.CardInfo{
			NormalizedURL: r.NormalizedURL,
			DirectURL:     r.DirectURL,
			PlatformID:    r.InfoData.PlatformID,
			CharacterID:   r.CharacterID,
			Name:          r.Name,
			Title:         r.Title,
			Tagline:       r.Tagline,
			CreateTime:    r.CreateTime,
			UpdateTime:    r.UpdateTime,
			IsForked:      r.IsFork,
			Tags: slicesx.Map(r.Tags, func(tag Tag) models.Tag {
				return models.Tag{
					Slug: models.Slug(tag.ID),
					Name: tag.Name,
				}
			}),
		},
		CreatorInfo: models.CreatorInfo{
			Nickname:   r.Creator.Nickname,
			Username:   r.Creator.Username,
			PlatformID: r.Creator.PlatformID,
		},
		BookUpdateTime: r.BookUpdateTime,
		GreetingsCount: r.GreetingsCount,
	}
}

// ImportHeader is a record header with import data
type ImportHeader struct {
	ID RID
	ImportData
}

// ImportData contains import-related metadata
type ImportData struct {
	ImportTime  timestamp.Nano
	ImportIndex int
}

// InfoHeader is a record header with info data
type InfoHeader struct {
	ID RID
	InfoData
}

// InfoData contains the core information about a character card
type InfoData struct {
	Source         source.ID
	NormalizedURL  string
	DirectURL      string
	PlatformID     string
	CharacterID    string
	Name           string
	Title          string
	Tagline        string
	CreateTime     timestamp.Nano
	UpdateTime     timestamp.Nano
	BookUpdateTime timestamp.Nano
	GreetingsCount int
	IsFork         bool
	Tags           []Tag
}

// LatestUpdateTime returns the most recent update timestamp
func (r *InfoData) LatestUpdateTime() timestamp.Nano {
	return max(r.UpdateTime, r.BookUpdateTime)
}

// SyncHeader is a record header with sync data
type SyncHeader struct {
	ID RID
	SyncData
}

// SyncData contains synchronization status and timestamp
type SyncData struct {
	SyncTime   timestamp.Nano
	SyncStatus SyncStatus
}

// ExportHeader is a record header with export data
type ExportHeader struct {
	ID RID
	ExportData
}

// ExportData contains export-related metadata
type ExportData struct {
	ExportTime      timestamp.Nano
	ExportedVersion timestamp.Nano
}

// AuxHeader is a record header with auxiliary data
type AuxHeader struct {
	ID RID
	AuxData
}

// AuxData contains auxiliary flags like favorite status
type AuxData struct {
	Favorite bool
}

// Creator represents the creator of a character card
type Creator struct {
	ID         CID `json:"CID"`
	Nickname   string
	Username   string
	PlatformID string    `json:"CreatorPlatformID"`
	Source     source.ID `json:"CreatorSource"`
}

// Tag represents a tag associated with a character card
type Tag struct {
	ID   TID
	Name string
}

// TagNames extracts the names from a slice of tags
func TagNames(tags []Tag) []string {
	return slicesx.Map(tags, func(tag Tag) string {
		return tag.Name
	})
}

// TagIDs extracts the IDs from a slice of tags
func TagIDs(tags []Tag) []TID {
	return slicesx.Map(tags, func(tag Tag) TID {
		return tag.ID
	})
}
