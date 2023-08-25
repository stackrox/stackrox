package postgres

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
)

func scopeContextToQuery(ctx context.Context, q *v1.Query) (*v1.Query, error) {
	scopeQ, err := scoped.GetQueryForAllScopes(ctx)
	if err != nil {
		return nil, err
	}
	if scopeQ == nil {
		return q, nil
	}

	return cloneAndCombine(q, scopeQ), nil
}

func cloneAndCombine(q *v1.Query, scopeQ *v1.Query) *v1.Query {
	if q == nil {
		return scopeQ
	}
	if scopeQ == nil {
		return q
	}

	// Select, Group By, and Pagination must be set on the top-level query to be picked up by the query parser.
	// Therefore, move them to the top-level query.

	cloned := q.Clone()
	selects := cloned.GetSelects()
	groupBy := cloned.GetGroupBy()
	pagination := cloned.GetPagination()

	// Removing this from to-be nested query is optional because selects, group by and pagination from
	// the nested query is ignored anyway. However, this make it safer.
	cloned.Selects = nil
	cloned.GroupBy = nil
	cloned.Pagination = nil

	cloned = searchPkg.ConjunctionQuery(cloned, scopeQ)
	cloned.Selects = selects
	cloned.GroupBy = groupBy
	cloned.Pagination = pagination
	return cloned
}
