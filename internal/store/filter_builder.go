package store

import (
	"fmt"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/internal/ent/card"
	"github.com/r3dpixel/card-client/internal/ent/predicate"
	"github.com/r3dpixel/card-client/internal/ent/schema"
	"github.com/r3dpixel/card-client/services/filter"
)

type textPredicateBuilder func(tf filter.TextFilter) (predicate.Card, error)

type booleanPredicateBuilder func(bf filter.BooleanFilter) (predicate.Card, error)

type EntFilterBuilder struct {
	textPredicateBuilders    map[filter.PublicField]textPredicateBuilder
	booleanPredicateBuilders map[filter.PublicField]booleanPredicateBuilder
	sortableFields           map[filter.PublicField]string
}

func NewFilterBuilder() EntFilterBuilder {
	s := EntFilterBuilder{}

	s.textPredicateBuilders = map[filter.PublicField]textPredicateBuilder{
		filter.FieldCardName:      buildGenericTextPredicate(card.CardNameContainsFold, card.CardNameContains),
		filter.FieldCharacterName: buildGenericTextPredicate(card.CharacterNameContainsFold, card.CharacterNameContains),
		filter.FieldCreator:       buildGenericTextPredicate(card.CreatorContainsFold, card.CreatorContains),
	}

	s.booleanPredicateBuilders = map[filter.PublicField]booleanPredicateBuilder{
		filter.FieldFavorite: func(bf filter.BooleanFilter) (predicate.Card, error) {
			return card.FavoriteEQ(bf.Value), nil
		},
	}

	s.sortableFields = map[filter.PublicField]string{
		filter.FieldUpdatedTime:  schema.FieldCardUpdateTime,
		filter.FieldCreatedTime:  schema.FieldCardCreateTime,
		filter.FieldImportedTime: schema.FieldCardImportTime,
	}

	return s
}

func buildGenericTextPredicate(
	containsFold func(string) predicate.Card,
	contains func(string) predicate.Card,
) textPredicateBuilder {
	return func(tf filter.TextFilter) (predicate.Card, error) {
		//if tf.MatchMode.Regex {
		//	re, err := regexp.Compile(tf.Value)
		//	if err != nil {
		//		return nil, fmt.Errorf("invalid regex: %w", err)
		//	}
		//	return matches(re), nil
		//}
		if tf.MatchMode.CaseSensitive {
			return contains(tf.Value), nil
		}
		return containsFold(tf.Value), nil
	}
}

func (s *EntFilterBuilder) ApplyFilter(query *ent.CardQuery, f filter.SearchFilter) (*ent.CardQuery, error) {
	var predicates []predicate.Card

	for _, tf := range f.TextFilter {
		builder, ok := s.textPredicateBuilders[tf.Field]
		if !ok {
			return nil, fmt.Errorf("unsupported text filter field: %s", tf.Field)
		}
		p, err := builder(tf)
		if err != nil {
			return nil, fmt.Errorf("failed to build text filter for field %s: %w", tf.Field, err)
		}
		predicates = append(predicates, p)
	}

	for _, bf := range f.BooleanFilters {
		builder, ok := s.booleanPredicateBuilders[bf.Field]
		if !ok {
			return nil, fmt.Errorf("unsupported boolean filter field: %s", bf.Field)
		}
		p, err := builder(bf)
		if err != nil {
			return nil, fmt.Errorf("failed to build boolean filter for field %s: %w", bf.Field, err)
		}
		predicates = append(predicates, p)
	}

	if len(f.Sources) > 0 {
		predicates = append(predicates, card.SourceIn(f.Sources...))
	}

	if len(f.Statuses) > 0 {
		predicates = append(predicates, card.LastUpdateStatusIn(f.Statuses...))
	}

	if len(predicates) > 0 {
		query = query.Where(predicates...)
	}

	for _, sf := range f.SortFilters {
		orderFuncs, err := s.buildSortOrder(sf)
		if err != nil {
			return nil, err // Error is already descriptive.
		}
		query = query.Order(orderFuncs...)
	}

	return query, nil
}

func (s *EntFilterBuilder) buildSortOrder(sf filter.SortFilter) ([]card.OrderOption, error) {
	column, ok := s.sortableFields[sf.Field]
	if !ok {
		return nil, fmt.Errorf("unsupported sortable field: %s", sf.Field)
	}

	var primaryOrder card.OrderOption
	if sf.Direction == filter.DESCENDING {
		primaryOrder = ent.Desc(column)
	} else {
		primaryOrder = ent.Asc(column)
	}

	if sf.Field == filter.FieldImportedTime {
		if sf.Direction == filter.DESCENDING {
			return []card.OrderOption{primaryOrder, ent.Desc(card.FieldBatchOrder)}, nil
		} else {
			return []card.OrderOption{primaryOrder, ent.Asc(card.FieldBatchOrder)}, nil
		}
	}

	return []card.OrderOption{primaryOrder}, nil
}
