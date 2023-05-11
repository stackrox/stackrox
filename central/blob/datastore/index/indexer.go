package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the interface for indexing blobs
//
//go:generate mockgen-wrapper
type Indexer interface {
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
