package search

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var _ Searcher = (*searcherImpl)(nil)

type searcherImpl struct {
	indexer index.Indexer
}

// Search queries for events and returns the search results.
func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.indexer.Search(ctx, q)
}

// Count returns the number of search results from the query.
func (s *searcherImpl) Count(ctx context.Context, query *v1.Query) (int, error) {
	return s.indexer.Count(ctx, query)
}
