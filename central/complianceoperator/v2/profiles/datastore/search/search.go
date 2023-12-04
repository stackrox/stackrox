package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher is scan configuration searcher
type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage pgStore.Store, search search.Searcher) Searcher {
	return &searcherImpl{
		storage:  storage,
		searcher: search,
	}
}
