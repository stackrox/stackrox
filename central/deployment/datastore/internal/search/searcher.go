package search

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing alerts
//
//go:generate mockgen-wrapper
type Searcher interface {
	SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error)
	SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error)

	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}
