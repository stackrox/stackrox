package search

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

var _ Searcher = (*searcherImpl)(nil)

type searcherImpl struct {
	indexer index.Indexer
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, query *v1.Query) (int, error) {
	return ds.indexer.Count(ctx, query)
}
