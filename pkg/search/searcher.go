package search

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher allows you to search objects.
//go:generate mockgen-wrapper Searcher
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]Result, error)
}

type searcherFunc func(ctx context.Context, q *v1.Query) ([]Result, error)

// Search runs the search function on the input context and query.
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

// WrapSearchFunc a function that takes in a context and query and produces results as a Searcher.
func WrapSearchFunc(f func(ctx context.Context, q *v1.Query) ([]Result, error)) Searcher {
	return searcherFunc(func(ctx context.Context, q *v1.Query) ([]Result, error) {
		return f(ctx, q)
	})
}

// UnsafeSearcher executes an input query without checking permissions for returned objects.
type UnsafeSearcher interface {
	Search(q *v1.Query) ([]Result, error)
}

type unsafeSearcherFunc func(q *v1.Query) ([]Result, error)

// Search runs the search function on the input context and query.
func (f unsafeSearcherFunc) Search(q *v1.Query) ([]Result, error) {
	return f(q)
}

// WrapUnsafeSearchFunc takes in an unsafe search executing function and returns it as an UnsafeSearcher implementation.
func WrapUnsafeSearchFunc(f func(q *v1.Query) ([]Result, error)) UnsafeSearcher {
	return unsafeSearcherFunc(func(q *v1.Query) ([]Result, error) {
		return f(q)
	})
}
