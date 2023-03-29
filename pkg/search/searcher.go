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

// FuncSearcher is a collection of functions that implements the searcher interface.
type FuncSearcher struct {
	SearchFunc func(ctx context.Context, q *v1.Query) ([]Result, error)
	CountFunc  func(ctx context.Context, q *v1.Query) (int, error)
}

// Search runs the search function on the input context and query.
func (f FuncSearcher) Search(ctx context.Context, q *v1.Query) ([]Result, error) {
	return f.SearchFunc(ctx, q)
}

// Count runs the count function on the input context and query.
func (f FuncSearcher) Count(ctx context.Context, q *v1.Query) (int, error) {
	return f.CountFunc(ctx, q)
}
