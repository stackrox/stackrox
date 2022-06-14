package blevesearch

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// UnsafeSearcher is a searcher that does not take in a context to perform SAC enforcement.
//go:generate mockgen-wrapper
type UnsafeSearcher interface {
	Search(q *v1.Query, opts ...SearchOption) ([]search.Result, error)
	Count(q *v1.Query, opts ...SearchOption) (int, error)
}

// UnsafeSearcherImpl is a search function that does not take in a context to perform SAC enforcement.
type UnsafeSearcherImpl struct {
	SearchFunc func(q *v1.Query, opts ...SearchOption) ([]search.Result, error)
	CountFunc  func(q *v1.Query, opts ...SearchOption) (int, error)
}

// Search implements Search of `UnsafeSearcher` interface.
func (f UnsafeSearcherImpl) Search(q *v1.Query, opts ...SearchOption) ([]search.Result, error) {
	return f.SearchFunc(q, opts...)
}

// Count implements Count of `UnsafeSearcher` interface.
func (f UnsafeSearcherImpl) Count(q *v1.Query, opts ...SearchOption) (int, error) {
	return f.CountFunc(q, opts...)
}

// WrapUnsafeSearcherAsSearcher wraps an unsafe searcher not taking a context parameter in its Search method
// into the Searcher interface.
// CAUTION: Only use this function if you exactly know what you are doing; otherwise you risk data leakage.
func WrapUnsafeSearcherAsSearcher(searcher UnsafeSearcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(_ context.Context, q *v1.Query) ([]search.Result, error) {
			return searcher.Search(q)
		},
		CountFunc: func(_ context.Context, q *v1.Query) (int, error) {
			return searcher.Count(q)
		},
	}
}
