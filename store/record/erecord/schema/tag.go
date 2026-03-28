package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/digital-foxy/card-client/store/resource"
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
			GoType(resource.EmptyTID).
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
