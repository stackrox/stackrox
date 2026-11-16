package search

import v1 "github.com/stackrox/rox/generated/api/v1"

// MatchFieldSubQuery filters records whose field value matches a value returned
// by query in category. For example, it can filter images by matching their IDs
// against the image IDs used by deployments in a particular namespace.
//
// The query must select one field containing a single value per record, such as
// an ID or name. Arrays, maps and computed fields (such as counts) are unsupported,
// as are grouping, filters on selected fields or aggregates, and sorting by
// computed fields. The SQL search backend checks access to category separately.
func MatchFieldSubQuery(field FieldLabel, category v1.SearchCategory, query *v1.Query) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_MatchFieldSubQuery{MatchFieldSubQuery: &v1.MatchFieldSubQuery{
			Field: field.String(), Category: category, Query: query,
		}},
	})
}
