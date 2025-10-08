package entrecord

import (
	"github.com/r3dpixel/card-client/store/record/entrecord/ent"
	"github.com/r3dpixel/card-client/store/record/entrecord/ent/predicate"
	"github.com/r3dpixel/card-client/store/record/entrecord/ent/recordentity"
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/structx"
)

type stringPredicate func(v string) predicate.RecordEntity
type boolPredicate func(v bool) predicate.RecordEntity

type textPredicate struct {
	containsFold stringPredicate
	contains     stringPredicate
}

type filterBuilder struct {
	textPredicates    map[resource.Field]textPredicate
	booleanPredicates map[resource.Field]boolPredicate
	sortableFields    map[resource.Field]struct{}
}

func newFilterBuilder() *filterBuilder {
	s := &filterBuilder{}

	s.textPredicates = map[resource.Field]textPredicate{
		resource.FieldRecordTitle: {containsFold: recordentity.TitleContainsFold, contains: recordentity.TitleContains},
		resource.FieldRecordName:  {containsFold: recordentity.NameContainsFold, contains: recordentity.NameContains},
		//resource.FieldRecordCreatorID: {containsFold: recordentity.CreatorIDContainsFold, contains: recordentity.CreatorContains},
	}

	s.booleanPredicates = map[resource.Field]boolPredicate{
		resource.FieldRecordFavorite: recordentity.FavoriteEQ,
	}

	s.sortableFields = map[resource.Field]struct{}{
		resource.FieldRecordImportTime: structx.Empty,
		resource.FieldRecordUpdateTime: structx.Empty,
		resource.FieldRecordCreateTime: structx.Empty,
	}

	return s
}

func (s *filterBuilder) applyTextPredicate(tf resource.TextFilter) predicate.RecordEntity {
	//if tf.MatchMode.Regex {
	//	re, err := regexp.Compile(tf.Value)
	//	if err != nil {
	//		return nil, fmt.Errorf("invalid regex: %w", err)
	//	}
	//	return matches(re), nil
	//}
	if tf.MatchMode.CaseSensitive {
		return s.textPredicates[tf.Field].contains(tf.Value)
	}
	return s.textPredicates[tf.Field].containsFold(tf.Value)

}

func (s *filterBuilder) ApplyFilter(query *ent.RecordEntityQuery, f resource.Filter) *ent.RecordEntityQuery {
	var predicates []predicate.RecordEntity

	for _, tf := range f.TextFilter {
		predicates = append(predicates, s.applyTextPredicate(tf))
	}

	for _, bf := range f.BooleanFilters {
		predicates = append(predicates, s.booleanPredicates[bf.Field](bf.Value))
	}

	if len(f.Sources) > 0 {
		predicates = append(predicates, recordentity.SourceIn(f.Sources...))
	}

	if len(f.Statuses) > 0 {
		predicates = append(predicates, recordentity.SyncStatusIn(f.Statuses...))
	}

	if len(predicates) > 0 {
		query = query.Where(predicates...)
	}

	for _, sf := range f.SortFilters {
		query = query.Order(s.buildSortOrder(sf)...)
	}

	return query
}

func (s *filterBuilder) buildSortOrder(sf resource.SortFilter) []recordentity.OrderOption {
	if _, ok := s.sortableFields[sf.Field]; !ok {
		return nil
	}

	var primaryOrder recordentity.OrderOption
	if sf.Direction == resource.DESCENDING {
		primaryOrder = ent.Desc(sf.Field)
	} else {
		primaryOrder = ent.Asc(sf.Field)
	}

	if sf.Field == resource.FieldRecordImportTime {
		if sf.Direction == resource.DESCENDING {
			return []recordentity.OrderOption{primaryOrder, ent.Desc(recordentity.FieldImportIndex)}
		} else {
			return []recordentity.OrderOption{primaryOrder, ent.Asc(recordentity.FieldImportIndex)}
		}
	}

	return []recordentity.OrderOption{primaryOrder}
}
