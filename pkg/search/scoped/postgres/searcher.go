package postgres

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

// WithScoping allows the input searcher to be scoped.
func WithScoping(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			scopes, hasScope := scoped.GetAllScopes(ctx)
			if hasScope {
				var err error
				q, err = scopeQuery(q, scopes)
				if err != nil || q == nil {
					return nil, err
				}
			}
			return searcher.Search(ctx, q)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			scopes, hasScope := scoped.GetAllScopes(ctx)
			if hasScope {
				var err error
				q, err = scopeQuery(q, scopes)
				if err != nil || q == nil {
					return 0, err
				}
			}
			return searcher.Count(ctx, q)
		},
	}
}

func scopeQuery(q *v1.Query, scopes []scoped.Scope) (*v1.Query, error) {
	pagination := q.GetPagination()
	q.Pagination = nil
	conjuncts := []*v1.Query{q}
	for _, scope := range scopes {
		schema := mapping.GetTableFromCategory(scope.Level)
		if schema == nil {
			utils.Should(errors.Errorf("no schema registered for search category %s", scope.Level))
			return q, nil
		}
		idField := schema.ID()
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddExactMatches(search.FieldLabel(idField.Search.FieldName), scope.ID).ProtoQuery())
	}
	ret := search.ConjunctionQuery(conjuncts...)
	ret.Pagination = pagination
	return ret, nil
}
