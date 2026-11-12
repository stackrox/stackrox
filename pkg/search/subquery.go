package search

import v1 "github.com/stackrox/rox/generated/api/v1"

// MatchFieldSubQuery matches field against the values selected by query in category.
// The query must select exactly one scalar field using the usual query selects.
// Subqueries are supported by the SQL search backend, including its access checks.
func MatchFieldSubQuery(field FieldLabel, category v1.SearchCategory, query *v1.Query) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_MatchFieldSubQuery{MatchFieldSubQuery: &v1.MatchFieldSubQuery{
			Field: field.String(), Category: category, Query: query,
		}},
	})
}
