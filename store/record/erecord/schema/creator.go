package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/card-fetcher/source"
	"github.com/r3dpixel/toolkit/stringsx"
)

type CreatorEntity struct {
	ent.Schema
}

func (CreatorEntity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: resource.ContainerCreators},
	}
}

func (CreatorEntity) Fields() []ent.Field {
	return []ent.Field{
		field.String(resource.FieldCreatorID).
			GoType(resource.CID(stringsx.Empty)).
			Immutable().
			Comment("ID of the creator"),
		field.String(resource.FieldCreatorNickname).
			Comment("Nickname of the creator"),
		field.String(resource.FieldCreatorUsername).
			Comment("Username of the creator"),
		field.String(resource.FieldCreatorPlatformID).
			Comment("Platform ID of the creator"),
		field.Enum(resource.FieldCreatorSource).
			GoType(source.ChubAI).
			Comment("SOURCE (ChubAI, WyvernChat, etc...)"),
	}
}

func (CreatorEntity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From(resource.ContainerRecords, RecordEntity.Type).
			Ref(CreatorEdge),
	}
}
