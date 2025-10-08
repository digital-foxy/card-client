package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/stringsx"
)

type TagEntity struct {
	ent.Schema
}

func (TagEntity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: resource.ContainerTags},
	}
}

func (TagEntity) Fields() []ent.Field {
	return []ent.Field{
		field.String(resource.FieldTagID).
			GoType(resource.TID(stringsx.Empty)).
			Immutable().
			Comment("Slug of the tag"),
		field.String(resource.FieldTagName).
			Comment("Tag name"),
	}
}

func (TagEntity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From(resource.ContainerRecords, RecordEntity.Type).
			Ref(resource.ContainerTags),
	}
}
