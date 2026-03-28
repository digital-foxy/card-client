package erecord

import (
	"fmt"
	"regexp"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/digital-foxy/card-client/store/record/erecord/ent"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/creatorentity"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/predicate"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/recordentity"
	"github.com/digital-foxy/card-client/store/record/erecord/ent/tagentity"
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/toolkit/structx"
)

func recordRegexPredicate(pattern string, field string) predicate.RecordEntity {
	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString(field).WriteString(" REGEXP ").Arg(pattern)
		}))
	}
}

func creatorRegexPredicate(pattern string) predicate.RecordEntity {
	return recordentity.HasCreatorWith(func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString(creatorentity.FieldNickname).WriteString(" REGEXP ").Arg(pattern)
		}))
	})
}

type stringPredicate func(v string) predicate.RecordEntity
type boolPredicate func(v bool) predicate.RecordEntity

type textPredicate struct {
	containsFold stringPredicate
	regex        stringPredicate
}

// filterBuilder builds SQL predicates from resource.Filter
type filterBuilder struct {
	textPredicates    map[resource.Field]textPredicate
	booleanPredicates map[resource.Field]boolPredicate
	sortableFields    map[resource.Field]struct{}
}

func newFilterBuilder() *filterBuilder {
	s := &filterBuilder{}

	s.textPredicates = map[resource.Field]textPredicate{
		resource.FieldRecordTitle: {
			containsFold: recordentity.TitleContainsFold,
			regex:        func(v string) predicate.RecordEntity { return recordRegexPredicate(v, recordentity.FieldTitle) },
		},
		resource.FieldRecordName: {
			containsFold: recordentity.NameContainsFold,
			regex:        func(v string) predicate.RecordEntity { return recordRegexPredicate(v, recordentity.FieldName) },
		},
		resource.FieldCreatorNickname: {
			containsFold: func(v string) predicate.RecordEntity {
				return recordentity.HasCreatorWith(creatorentity.NicknameContainsFold(v))
			},
			regex: creatorRegexPredicate,
		},
	}

	s.booleanPredicates = map[resource.Field]boolPredicate{
		resource.FieldRecordFavorite: recordentity.FavoriteEQ,
		resource.FieldRecordIsFork:   recordentity.IsForkEQ,
		resource.FieldRecordHasBook: func(v bool) predicate.RecordEntity {
			if v {
				return recordentity.BookUpdateTimeNEQ(0)
			}
			return recordentity.BookUpdateTimeEQ(0)
		},
	}

	s.sortableFields = map[resource.Field]struct{}{
		resource.FieldRecordImportTime: structx.Empty,
		resource.FieldRecordUpdateTime: structx.Empty,
		resource.FieldRecordCreateTime: structx.Empty,
		resource.FieldRecordSyncTime:   structx.Empty,
	}

	return s
}

func (s *filterBuilder) applyTextPredicate(tf resource.TextFilter) predicate.RecordEntity {
	// If ANY toggle is on, use regex
	if tf.MatchMode.CaseSensitive || tf.MatchMode.Regex || tf.MatchMode.WholeWord {
		pattern := tf.Value

		// If not regex mode, escape special regex chars for literal matching
		if !tf.MatchMode.Regex {
			pattern = regexp.QuoteMeta(pattern)
		}

		// Add word boundaries if whole word mode
		if tf.MatchMode.WholeWord {
			pattern = fmt.Sprintf("\\b%s\\b", pattern)
		}

		// Add case-insensitive flag if needed
		if !tf.MatchMode.CaseSensitive {
			pattern = "(?i)" + pattern
		}

		return s.textPredicates[tf.Field].regex(pattern)
	}

	// All toggles off - use case-insensitive contains
	return s.textPredicates[tf.Field].containsFold(tf.Value)
}

func (s *filterBuilder) applyTagFilter(tf resource.TagFilter) predicate.RecordEntity {
	if len(tf.Names) == 0 {
		return nil
	}

	if tf.Mode == resource.ANY {
		// ANY mode: record has at least one of the tags
		return recordentity.HasTagsWith(tagentity.NameIn(tf.Names...))
	}

	// ALL mode: record must have all specified tags
	var tagPredicates []predicate.RecordEntity
	for _, name := range tf.Names {
		tagPredicates = append(tagPredicates, recordentity.HasTagsWith(tagentity.NameEQ(name)))
	}
	return recordentity.And(tagPredicates...)
}

func (s *filterBuilder) applyContentFilter(cf resource.ContentFilter) predicate.RecordEntity {
	if len(cf.Values) == 0 || len(cf.Fields) == 0 {
		return nil
	}

	// Build value query (same for all fields, based on ValueMode)
	var valueQueryOperator string
	if cf.ValueMode == resource.ALL {
		valueQueryOperator = " AND "
	} else {
		valueQueryOperator = " OR "
	}

	// Build field-level queries
	var fieldQueries []string
	for _, field := range cf.Fields {
		// Build value queries for this field
		var valueQueries []string
		for _, value := range cf.Values {
			escapedValue := `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
			valueQueries = append(valueQueries, fmt.Sprintf("%s: %s", field, escapedValue))
		}

		// Combine values for this field
		fieldQuery := strings.Join(valueQueries, valueQueryOperator)

		// Wrap in parentheses if there are multiple values
		if len(valueQueries) > 1 {
			fieldQuery = "(" + fieldQuery + ")"
		}
		fieldQueries = append(fieldQueries, fieldQuery)
	}

	// Combine fields with FieldMode (ALL/ANY)
	var matchExpr string
	if cf.FieldMode == resource.ALL {
		// ALL fields must match the value condition
		matchExpr = strings.Join(fieldQueries, " AND ")
	} else {
		// ANY field must match the value condition
		matchExpr = strings.Join(fieldQueries, " OR ")
	}

	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString("id IN (SELECT id FROM fts WHERE fts MATCH ").Arg(matchExpr).WriteString(")")
		}))
	}
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

	for _, cf := range f.ContentFilters {
		if p := s.applyContentFilter(cf); p != nil {
			predicates = append(predicates, p)
		}
	}

	if p := s.applyTagFilter(f.TagFilter); p != nil {
		predicates = append(predicates, p)
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
