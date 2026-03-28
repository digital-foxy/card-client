package resource

// Field is an alias for string representing a database field name
type Field = string

// Record container and field constants
const (
	ContainerRecords           string = "records"
	FieldRecordID              Field  = "id"
	FieldRecordImportTime      Field  = "import_time"
	FieldRecordImportIndex     Field  = "import_index"
	FieldRecordSource          Field  = "source"
	FieldRecordNormalizedURL   Field  = "normalized_url"
	FieldRecordDirectURL       Field  = "direct_url"
	FieldRecordPlatformID      Field  = "platform_id"
	FieldRecordCharacterID     Field  = "character_id"
	FieldRecordName            Field  = "name"
	FieldRecordTitle           Field  = "title"
	FieldRecordCreatorID       Field  = "creator_id"
	FieldRecordTagline         Field  = "tagline"
	FieldRecordCreateTime      Field  = "create_time"
	FieldRecordUpdateTime      Field  = "update_time"
	FieldRecordBookUpdateTime  Field  = "book_update_time"
	FieldGreetingsCount        Field  = "greetings_count"
	FieldRecordHasBook         Field  = "has_book"
	FieldRecordIsFork          Field  = "is_fork"
	FieldRecordSyncTime        Field  = "sync_time"
	FieldRecordSyncStatus      Field  = "sync_status"
	FieldRecordExportTime      Field  = "export_time"
	FieldRecordExportedVersion Field  = "exported_version"
	FieldRecordFavorite        Field  = "favorite"
)

// Creator container and field constants
const (
	ContainerCreators      string = "creators"
	FieldCreatorID         Field  = "id"
	FieldCreatorNickname   Field  = "nickname"
	FieldCreatorUsername   Field  = "username"
	FieldCreatorPlatformID Field  = "platform_id"
	FieldCreatorSource     Field  = "source"
)

// Tag container and field constants
const (
	ContainerTags string = "tags"
	FieldTagID    Field  = "id"
	FieldTagName  Field  = "name"
)

// Content field constants for character card data
const (
	FieldContentDescription             Field = "description"
	FieldContentPersonality             Field = "personality"
	FieldContentScenario                Field = "scenario"
	FieldContentFirstMes                Field = "first_mes"
	FieldContentMesExample              Field = "mes_example"
	FieldContentCreatorNotes            Field = "creator_notes"
	FieldContentSystemPrompt            Field = "system_prompt"
	FieldContentPostHistoryInstructions Field = "post_history_instructions"
	FieldContentAlternateGreetings      Field = "alternate_greetings"
	FieldContentDepthPrompt             Field = "depth_prompt"
)

// ContentFields is a list of all content field names
var ContentFields = []Field{
	FieldContentDescription,
	FieldContentPersonality,
	FieldContentScenario,
	FieldContentFirstMes,
	FieldContentMesExample,
	FieldContentCreatorNotes,
	FieldContentSystemPrompt,
	FieldContentPostHistoryInstructions,
	FieldContentAlternateGreetings,
	FieldContentDepthPrompt,
}
