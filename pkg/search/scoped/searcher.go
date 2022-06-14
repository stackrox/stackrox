package scoped

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/stackrox/pkg/search"
)

// TransformationProvider provides a transformation.OneToMany for a given category.
type TransformationProvider interface {
	Get(v1.SearchCategory) transformation.OneToMany
}

// WithScoping allows the input searcher to be scoped.
// The output searcher will fetch the scope from the context, and apply it to the input searcher via limiting any
// searches to the items in the scope as defined by the input transformation map.
//
// For example, if the input scope is <image, sha:derp>, then we will use the Transformation provider to get a
// transformation for the 'image' search category. Then we will apply it to the id 'sha:derp'. This will give us a set
// if ids for the type we are currently searching. We need to limit the search to just those ids, so we add them as a
// set of DocIDs in a conjunction with the input query.
func WithScoping(searcher search.Searcher, provider TransformationProvider) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			scope, hasScope := GetScope(ctx)
			if hasScope {
				var err error
				q, err = scopeQuery(ctx, q, scope, provider)
				if err != nil || q == nil {
					return nil, err
				}
			}
			return searcher.Search(ctx, q)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			scope, hasScope := GetScope(ctx)
			if hasScope {
				var err error
				q, err = scopeQuery(ctx, q, scope, provider)
				if err != nil || q == nil {
					return 0, err
				}
			}
			return searcher.Count(ctx, q)
		},
	}
}

func scopeQuery(ctx context.Context, q *v1.Query, scope Scope, provider TransformationProvider) (*v1.Query, error) {
	dstTransformer := provider.Get(scope.Level)
	if dstTransformer == nil {
		return nil, errors.Errorf("no scope transformations registered for %s", scope.Level)
	}
	scopeIDs := dstTransformer(ctx, []byte(scope.ID))
	if len(scopeIDs) == 0 {
		return nil, nil
	}
	ret := search.ConjunctionQuery(q, search.NewQueryBuilder().AddDocIDs(toStr(scopeIDs...)...).ProtoQuery())
	ret.Pagination = q.Pagination
	return ret, nil
}

func toStr(keys ...[]byte) []string {
	ret := make([]string, 0, len(keys))
	for _, key := range keys {
		ret = append(ret, string(key))
	}
	return ret
}
