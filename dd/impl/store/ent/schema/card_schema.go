package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/timestamp"
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
			GoType(EmptyCardID).
			DefaultFunc(func() CardID { return CardID(uuid.NewString()) }).
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
