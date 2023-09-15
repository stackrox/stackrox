package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Indexer is the notifications indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
}
