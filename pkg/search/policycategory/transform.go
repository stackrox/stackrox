package policycategory

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// TransformCategoryNameFieldsQuery transforms category name fields for the new data layout for categories in postgres.
func TransformCategoryNameFieldsQuery(q *v1.Query) *v1.Query {
	// Local copy to avoid changing input.
	local := q.CloneVT()
	pagination := local.GetPagination()
	local.Pagination = nil

	handleCategoryNameQuery(local)

	local.Pagination = pagination
	return local
}

func handleCategoryNameQuery(q *v1.Query) {
	if q.GetQuery() == nil {
		return
	}

	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.Category.String() {
			matchFieldQuery.MatchFieldQuery.Field = search.PolicyCategoryName.String()
		}
	})
}
