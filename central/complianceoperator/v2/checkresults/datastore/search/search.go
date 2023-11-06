package search

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/index"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher is searcher for check results
type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage pgStore.Store, indexer index.Indexer, search search.Searcher) *searcherImpl {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: search,
	}
}
