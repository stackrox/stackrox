package search

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher allows you to search objects.
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]Result, error)
}

type searcherFunc func(ctx context.Context, q *v1.Query) ([]Result, error)

func (f searcherFunc) Search(ctx context.Context, q *v1.Query) ([]Result, error) {
	return f(ctx, q)
}

// WrapContextLessSearcher wraps a searcher not taking a context parameter in its Search method
// into the Searcher interface.
func WrapContextLessSearcher(searcher interface {
	Search(q *v1.Query) ([]Result, error)
}) Searcher {
	return searcherFunc(func(_ context.Context, q *v1.Query) ([]Result, error) {
		return searcher.Search(q)
	})
}
