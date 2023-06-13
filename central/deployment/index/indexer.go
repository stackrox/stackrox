package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer encapsulates the deployment indexer
//
//go:generate mockgen-wrapper
type Indexer interface {
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
}
