package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/toolkit/timestamp"
)

type RecordEntity struct {
	ent.Schema
}

func (RecordEntity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: resource.ContainerRecords,
		},
	}
}

func (RecordEntity) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64(resource.FieldRecordID).
			GoType(resource.RID(0)).
			Immutable().
			Comment("Auto-increment resource ID"),
		field.Int(resource.FieldRecordImportIndex).
			Immutable().
			Comment("Import index inside batch"),
		field.Int64(resource.FieldRecordImportTime).
			Immutable().
			GoType(timestamp.Nano(0)).
			Comment("Import timestamp"),
		field.Enum(resource.FieldRecordSource).
			GoType(source.ChubAI).
			Comment("SOURCE (ChubAI, WyvernChat, etc...)"),
		field.String(resource.FieldRecordNormalizedURL).
			Comment("Normalized card URL"),
		field.String(resource.FieldRecordDirectURL).
			Comment("Direct card URL"),
		field.String(resource.FieldRecordPlatformID).
			Comment("Platform ID for card"),
		field.String(resource.FieldRecordCharacterID).
			Comment("Character identifier from platform"),
		field.String(resource.FieldRecordTitle).
			Comment("Card title"),
		field.String(resource.FieldRecordName).
			Comment("Character name"),
		field.String(resource.FieldRecordCreatorID).
			GoType(resource.EmptyCID).
			Comment("Card creator/author ID"),
		field.String(resource.FieldRecordTagline).
			Comment("Short description or tagline"),
		field.Int64(resource.FieldRecordCreateTime).
			GoType(timestamp.Nano(0)).
			Comment("Card creation timestamp"),
		field.Int64(resource.FieldRecordUpdateTime).
			GoType(timestamp.Nano(0)).
			Comment("Last update timestamp"),
		field.Bool(resource.FieldRecordIsFork).
			Default(false).
			Comment("Whether card is a fork"),
		field.Int64(resource.FieldRecordBookUpdateTime).
			GoType(timestamp.Nano(0)).
			Comment("Book/collection update timestamp"),
		field.Int(resource.FieldGreetingsCount).
			Default(-1).
			Comment("Number of alternate greetings"),
		field.Enum(resource.FieldRecordSyncStatus).
			GoType(resource.SyncSuccess).
			Comment("Synchronization status"),
		field.Int64(resource.FieldRecordSyncTime).
			GoType(timestamp.Nano(0)).
			Comment("Last synchronization timestamp"),
		field.Int64(resource.FieldRecordExportedVersion).
			GoType(timestamp.Nano(0)).
			Comment("Exported version timestamp"),
		field.Int64(resource.FieldRecordExportTime).
			GoType(timestamp.Nano(0)).
			Comment("Last export timestamp"),
		field.Bool(resource.FieldRecordFavorite).
			Comment("Whether card is marked as favorite"),
	}
}

func (RecordEntity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To(resource.ContainerTags, TagEntity.Type).
			StorageKey(edge.Table(RecordTagsContainer)),
		edge.To(CreatorEdge, CreatorEntity.Type).
			Field(resource.FieldRecordCreatorID).
			Required().
			Unique(),
	}
}

func (RecordEntity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields(resource.FieldRecordSource, resource.FieldRecordCharacterID).
			StorageKey(RecordCharacterIDIndex).
			Unique(),
		index.Fields(resource.FieldRecordSource, resource.FieldRecordPlatformID).
			StorageKey(RecordPlatformIDIndex).
			Unique(),
		index.Fields(resource.FieldRecordNormalizedURL).
			StorageKey(RecordNormalizedURLIndex).
			Unique(),
	}
}
