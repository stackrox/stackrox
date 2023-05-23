package policycategory

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// TransformCategoryNameFields transforms category name fields for the new data layout for categories in postgres.
func TransformCategoryNameFields(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			handleCategoryNameQuery(local)

			local.Pagination = pagination
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			handleCategoryNameQuery(local)

			local.Pagination = pagination
			return searcher.Count(ctx, local)
		},
	}
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
