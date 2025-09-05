package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	ischema "github.com/r3dpixel/card-client/services/scheme"
)

type Tag struct {
	ent.Schema
}

func (Tag) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: TableTags},
	}
}

func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.String(FieldTagID).
			GoType(ischema.TagID("")).
			Immutable(),
		field.String(FieldTagName),
	}
}

// Edges of the Tag.
func (Tag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From(TableCards, Card.Type).
			Ref(TableTags),
	}
}
