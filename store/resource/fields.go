package resource

type Field = string

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
	FieldRecordSyncTime        Field  = "sync_time"
	FieldRecordSyncStatus      Field  = "sync_status"
	FieldRecordExportTime      Field  = "export_time"
	FieldRecordExportedVersion Field  = "exported_version"
	FieldRecordFavorite        Field  = "favorite"
)

const (
	ContainerCreators      string = "creators"
	FieldCreatorID         Field  = "id"
	FieldCreatorNickname   Field  = "nickname"
	FieldCreatorUsername   Field  = "username"
	FieldCreatorPlatformID Field  = "platform_id"
	FieldCreatorSource     Field  = "source"
)

const (
	ContainerTags string = "tags"
	FieldTagID    Field  = "id"
	FieldTagName  Field  = "name"
)
