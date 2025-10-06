package search

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher allows you to search objects.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}
