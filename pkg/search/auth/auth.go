package auth

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

// WithAuthFilter filters searcher with authorization check for the specified resource
func WithAuthFilter(searcher search.Searcher, resource permissions.ResourceHandle) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			if !authChecker(ctx, resource).IsAllowed() {
				return nil, nil
			}
			return searcher.Search(ctx, q)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			if !authChecker(ctx, resource).IsAllowed() {
				return 0, nil
			}
			return searcher.Count(ctx, q)
		},
	}
}

func authChecker(ctx context.Context, resource permissions.ResourceHandle) sac.ScopeChecker {
	return sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(resource)
}
