package postgres

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/postgres/mapping"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/utils"
)

// WithScoping allows the input searcher to be scoped.
func WithScoping(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			scope, hasScope := scoped.GetScope(ctx)
			if hasScope {
				var err error
				q, err = scopeQuery(q, scope)
				if err != nil || q == nil {
					return nil, err
				}
			}
			return searcher.Search(ctx, q)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			scope, hasScope := scoped.GetScope(ctx)
			if hasScope {
				var err error
				q, err = scopeQuery(q, scope)
				if err != nil || q == nil {
					return 0, err
				}
			}
			return searcher.Count(ctx, q)
		},
	}
}

func scopeQuery(q *v1.Query, scope scoped.Scope) (*v1.Query, error) {
	schema := mapping.GetTableFromCategory(scope.Level)
	if schema == nil {
		utils.Should(errors.Errorf("no schema registered for search category %s", scope.Level))
		return q, nil
	}

	for _, f := range schema.Fields {
		// We only support ID in scope.
		if strings.ToLower(f.ColumnName) != "id" {
			continue
		}
		ret := search.ConjunctionQuery(q,
			search.NewQueryBuilder().AddExactMatches(search.FieldLabel(f.Search.FieldName), scope.ID).ProtoQuery(),
		)
		ret.Pagination = q.Pagination
		return ret, nil
	}
	return q, nil
}
