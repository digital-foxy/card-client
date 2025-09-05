package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	ischema "github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
)

const (
	TableCards                   string = "cards"
	FieldCardID                  string = "id"
	FieldBatchOrder              string = "batch_order"
	FieldCardSource              string = "source"
	FieldCardURL                 string = "card_url"
	FieldDirectURL               string = "direct_url"
	FieldCardName                string = "card_name"
	FieldCardCharacterName       string = "character_name"
	FieldCardPlatformID          string = "platform_id"
	FieldCardCharacterID         string = "character_id"
	FieldCardCreator             string = "creator"
	FieldCardTagline             string = "tagline"
	FieldCardCreateTime          string = "create_time"
	FieldCardUpdateTime          string = "update_time"
	FieldCardBookUpdateTime      string = "book_update_time"
	FieldCardCheckTime           string = "check_time"
	FieldCardImportTime          string = "import_time"
	FieldCardExportTime          string = "export_time"
	FieldCardLastExportedVersion string = "last_exported_version"
	FieldCardFavorite            string = "favorite"
	FieldCardLastUpdateStatus    string = "last_update_status"
)

type Card struct {
	ent.Schema
}

func (Card) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: TableCards},
	}
}

func (Card) Fields() []ent.Field {
	return []ent.Field{
		field.String(FieldCardID).
			GoType(ischema.EmptyCardID).
			DefaultFunc(func() ischema.CardID { return ischema.CardID(uuid.NewString()) }).
			Immutable(),
		field.Int(FieldBatchOrder).
			Default(0),
		field.Enum(FieldCardSource).
			GoType(source.ChubAI),
		field.String(FieldCardURL).
			Unique(),
		field.String(FieldDirectURL),
		field.String(FieldCardPlatformID),
		field.String(FieldCardCharacterID),
		field.String(FieldCardName),
		field.String(FieldCardCharacterName),
		field.String(FieldCardCreator),
		field.String(FieldCardTagline),
		field.Int64(FieldCardCreateTime).
			GoType(timestamp.Nano(0)),
		field.Int64(FieldCardUpdateTime).
			GoType(timestamp.Nano(0)),
		field.Int64(FieldCardBookUpdateTime).
			GoType(timestamp.Nano(0)).
			Default(0),
		field.Int64(FieldCardImportTime).
			GoType(timestamp.Nano(0)),
		field.Int64(FieldCardCheckTime).
			GoType(timestamp.Nano(0)),
		field.Enum(FieldCardLastUpdateStatus).
			GoType(ischema.UpdateSuccess),
		field.Int64(FieldCardExportTime).
			GoType(timestamp.Nano(0)).
			Default(0),
		field.Int64(FieldCardLastExportedVersion).
			GoType(timestamp.Nano(0)).
			Default(0),
		field.Bool(FieldCardFavorite).
			Default(false),
	}
}

func (Card) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To(TableTags, Tag.Type),
	}
}

func (Card) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields(FieldCardSource, FieldCardCharacterID).
			Unique(),
		index.Fields(FieldCardSource, FieldCardPlatformID).
			Unique(),
	}
}
