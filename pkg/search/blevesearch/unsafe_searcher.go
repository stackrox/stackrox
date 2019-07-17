package blevesearch

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// UnsafeSearcher is a searcher that does not take in a context to perform SAC enforcement.
type UnsafeSearcher interface {
	Search(q *v1.Query, opts ...SearchOption) ([]search.Result, error)
}

// UnsafeSearchFunc is a search function that does not take in a context to perform SAC enforcement.
type UnsafeSearchFunc func(q *v1.Query, opts ...SearchOption) ([]search.Result, error)

// Search implements the `UnsafeSearcher` interface.
func (f UnsafeSearchFunc) Search(q *v1.Query, opts ...SearchOption) ([]search.Result, error) {
	return f(q, opts...)
}

// WrapUnsafeSearcherAsSearcher wraps an unsafe searcher not taking a context parameter in its Search method
// into the Searcher interface.
// CAUTION: Only use this function if you exactly know what you are doing; otherwise you risk data leakage.
func WrapUnsafeSearcherAsSearcher(searcher UnsafeSearcher) search.Searcher {
	return search.Func(func(_ context.Context, q *v1.Query) ([]search.Result, error) {
		return searcher.Search(q)
	})
}
