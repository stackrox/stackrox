package search

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher allows you to search objects.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]Result, error)
}

// Func is a function that implements the searcher interface.
type Func func(ctx context.Context, q *v1.Query) ([]Result, error)

// Search runs the search function on the input context and query.
func (f Func) Search(ctx context.Context, q *v1.Query) ([]Result, error) {
	return f(ctx, q)
}
