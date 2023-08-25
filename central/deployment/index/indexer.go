package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	search "github.com/stackrox/rox/pkg/search"
)

// Indexer encapsulates the deployment indexer
//
//go:generate mockgen-wrapper
type Indexer interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}
