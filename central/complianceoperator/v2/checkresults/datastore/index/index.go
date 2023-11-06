package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Indexer is indexer for scan results
type Indexer interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
}
