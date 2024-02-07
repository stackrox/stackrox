package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher is compliance integrations searcher
type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage pgStore.Store) Searcher {
	return &searcherImpl{
		storage: storage,
	}
}
