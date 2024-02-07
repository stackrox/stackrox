package search

import (
	"context"

	"github.com/stackrox/rox/central/cloudsources/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

var _ Searcher = (*searcherImpl)(nil)

type searcherImpl struct {
	store store.Store
}

// Count returns the number of search results from the query.
func (s *searcherImpl) Count(ctx context.Context, query *v1.Query) (int, error) {
	return s.store.Count(ctx, query)
}
