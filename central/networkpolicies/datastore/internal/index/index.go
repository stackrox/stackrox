package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer allows for counting of network policies
type Indexer interface {
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
}
