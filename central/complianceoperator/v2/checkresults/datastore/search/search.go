package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing check results
type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage pgStore.Store, search search.Searcher) *searcherImpl {
	return &searcherImpl{
		storage:  storage,
		searcher: search,
	}
}
