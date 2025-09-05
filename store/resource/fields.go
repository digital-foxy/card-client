package resource

type Field string

const (
	ContainerRecords             string = "records"
	FieldCardID                  Field  = "id"
	FieldBatchOrder              Field  = "batch_order"
	FieldCardSource              Field  = "source"
	FieldCardURL                 Field  = "card_url"
	FieldDirectURL               Field  = "direct_url"
	FieldCardName                Field  = "card_name"
	FieldCardCharacterName       Field  = "character_name"
	FieldCardPlatformID          Field  = "platform_id"
	FieldCardCharacterID         Field  = "character_id"
	FieldCardCreator             Field  = "creator"
	FieldCardTagline             Field  = "tagline"
	FieldCardCreateTime          Field  = "create_time"
	FieldCardUpdateTime          Field  = "update_time"
	FieldCardBookUpdateTime      Field  = "book_update_time"
	FieldCardCheckTime           Field  = "check_time"
	FieldCardImportTime          Field  = "import_time"
	FieldCardExportTime          Field  = "export_time"
	FieldCardLastExportedVersion Field  = "last_exported_version"
	FieldCardFavorite            Field  = "favorite"
	FieldCardLastUpdateStatus    Field  = "last_update_status"
)

const (
	ContainerTags string = "tags"
	FieldTagID    Field  = "id"
	FieldTagName  Field  = "name"
)
