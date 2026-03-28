package resource

import (
	"testing"

	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/property"
	"github.com/stretchr/testify/assert"
)

// Helper to create a valid Record for testing
func createValidRecord() *Record {
	return &Record{
		ID: 1,
		ImportData: ImportData{
			ImportTime:  1000,
			ImportIndex: 1,
		},
		InfoData: InfoData{
			Source:         "chub",
			NormalizedURL:  "https://example.com/character/123",
			DirectURL:      "https://example.com/direct/123",
			PlatformID:     "platform-123",
			CharacterID:    "char-123",
			Name:           "Test Character",
			Title:          "Test Title",
			Tagline:        "Test tagline",
			CreateTime:     1_000_000_000,
			UpdateTime:     2_000_000_000,
			BookUpdateTime: 0,
			IsFork:         false,
			Tags: []Tag{
				{ID: "fantasy", Name: "fantasy"},
				{ID: "adventure", Name: "adventure"},
			},
		},
		Creator: Creator{
			ID:         "creator-1",
			Nickname:   "TestCreator",
			Username:   "test_user",
			PlatformID: "creator-123",
			Source:     "chub",
		},
		SyncData: SyncData{
			SyncTime:   3000,
			SyncStatus: SyncSuccess,
		},
		ExportData: ExportData{
			ExportTime:      4000,
			ExportedVersion: 5000,
		},
		AuxData: AuxData{
			Favorite: true,
		},
	}
}

// Helper to create a valid character.Sheet that matches the record
func createValidSheet() *character.Sheet {
	return &character.Sheet{
		Content: character.Content{
			SourceID:         "chub",
			CharacterID:      "char-123",
			PlatformID:       "platform-123",
			DirectLink:       "https://example.com/direct/123",
			Title:            "Test Title",
			Description:      "Test Description",
			Creator:          "TestCreator",
			Name:             "Test Character",
			Nickname:         "Test Character",
			CreatorNotes:     "Test tagline and more notes",
			Tags:             property.StringArray{"fantasy", "adventure"},
			CreationDate:     1,
			ModificationDate: 2,
		},
	}
}

func TestRecordIntegrity(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *Record
		hasOK     bool
	}{
		{
			name: "valid record has integrity",
			setupFunc: func() *Record {
				return createValidRecord()
			},
			hasOK: true,
		},
		{
			name: "missing source lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.InfoData.Source = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing normalized URL lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.NormalizedURL = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing direct URL lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.DirectURL = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing platform ID lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.InfoData.PlatformID = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing character ID lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.CharacterID = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing name lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.Name = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing title lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.Title = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "zero create time lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.CreateTime = 0
				return r
			},
			hasOK: false,
		},
		{
			name: "zero update time lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.UpdateTime = 0
				return r
			},
			hasOK: false,
		},
		{
			name: "missing nickname lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.Nickname = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing username lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.Username = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "missing creator platform ID lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.Creator.PlatformID = ""
				return r
			},
			hasOK: false,
		},
		{
			name: "whitespace source lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.InfoData.Source = "   "
				return r
			},
			hasOK: false,
		},
		{
			name: "whitespace normalized URL lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.NormalizedURL = "\t\n"
				return r
			},
			hasOK: false,
		},
		{
			name: "whitespace name lacks integrity",
			setupFunc: func() *Record {
				r := createValidRecord()
				r.Name = "  \t  "
				return r
			},
			hasOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := tt.setupFunc()
			got := record.Integrity()
			assert.Equal(t, tt.hasOK, got, "Record integrity check failed")
		})
	}
}

func TestFixIntegrity(t *testing.T) {
	tests := []struct {
		name           string
		recordFunc     func() *Record
		sheetFunc      func() *character.Sheet
		expectedStatus RecordIntegrity
	}{
		{
			name: "valid record and sheet - no fixes needed",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				return createValidSheet()
			},
			expectedStatus: OK,
		},
		{
			name: "nil sheet is broken",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				return nil
			},
			expectedStatus: BROKEN,
		},
		{
			name: "malformed record is broken",
			recordFunc: func() *Record {
				r := createValidRecord()
				r.InfoData.Source = ""
				return r
			},
			sheetFunc: func() *character.Sheet {
				return createValidSheet()
			},
			expectedStatus: BROKEN,
		},
		{
			name: "malformed sheet is broken",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.Description = ""
				return s
			},
			expectedStatus: BROKEN,
		},
		{
			name: "source mismatch - fixed",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.SourceID = property.String("different-source")
				return s
			},
			expectedStatus: FIXED,
		},
		{
			name: "character ID mismatch - fixed",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.CharacterID = property.String("different-char-id")
				return s
			},
			expectedStatus: FIXED,
		},
		{
			name: "creator mismatch - fixed",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.Creator = property.String("DifferentCreator")
				return s
			},
			expectedStatus: FIXED,
		},
		{
			name: "tagline not prefix is broken",
			recordFunc: func() *Record {
				r := createValidRecord()
				r.Tagline = "Test tagline"
				return r
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.Content.CreatorNotes = property.String("Different notes that don't start with tagline")
				return s
			},
			expectedStatus: BROKEN,
		},
		{
			name: "book update time without book - record fixed",
			recordFunc: func() *Record {
				r := createValidRecord()
				r.BookUpdateTime = 3000
				return r
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.CharacterBook = nil
				return s
			},
			expectedStatus: FIXED,
		},
		{
			name: "no book update time with book - record fixed",
			recordFunc: func() *Record {
				r := createValidRecord()
				r.BookUpdateTime = 0
				return r
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.CharacterBook = &character.Book{}
				return s
			},
			expectedStatus: FIXED,
		},
		{
			name: "tags mismatch - fixed",
			recordFunc: func() *Record {
				return createValidRecord()
			},
			sheetFunc: func() *character.Sheet {
				s := createValidSheet()
				s.Tags = property.StringArray{"fantasy", "scifi"}
				return s
			},
			expectedStatus: FIXED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := tt.recordFunc()
			sheet := tt.sheetFunc()
			status := record.FixIntegrity(sheet)
			assert.Equal(t, tt.expectedStatus, status, "Integrity status mismatch")
		})
	}
}

func TestToMetadata(t *testing.T) {
	record := createValidRecord()
	metadata := record.ToMetadata()

	// Verify all fields are correctly mapped
	assert.Equal(t, record.InfoData.Source, metadata.Source)
	assert.Equal(t, record.NormalizedURL, metadata.CardInfo.NormalizedURL)
	assert.Equal(t, record.DirectURL, metadata.CardInfo.DirectURL)
	assert.Equal(t, record.InfoData.PlatformID, metadata.CardInfo.PlatformID)
	assert.Equal(t, record.CharacterID, metadata.CardInfo.CharacterID)
	assert.Equal(t, record.Name, metadata.CardInfo.Name)
	assert.Equal(t, record.Title, metadata.CardInfo.Title)
	assert.Equal(t, record.Tagline, metadata.CardInfo.Tagline)
	assert.Equal(t, record.CreateTime, metadata.CardInfo.CreateTime)
	assert.Equal(t, record.UpdateTime, metadata.CardInfo.UpdateTime)
	assert.Equal(t, record.IsFork, metadata.CardInfo.IsForked)
	assert.Equal(t, record.BookUpdateTime, metadata.BookUpdateTime)

	// Verify creator fields
	assert.Equal(t, record.Creator.Nickname, metadata.CreatorInfo.Nickname)
	assert.Equal(t, record.Creator.Username, metadata.CreatorInfo.Username)
	assert.Equal(t, record.Creator.PlatformID, metadata.CreatorInfo.PlatformID)

	// Verify tags are correctly mapped
	assert.Equal(t, len(record.Tags), len(metadata.CardInfo.Tags))
	for i, tag := range record.Tags {
		assert.Equal(t, models.Slug(tag.ID), metadata.CardInfo.Tags[i].Slug)
		assert.Equal(t, tag.Name, metadata.CardInfo.Tags[i].Name)
	}
}
