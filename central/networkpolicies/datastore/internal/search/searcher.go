package search

import (
	"context"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher provides search functionality on existing network policies
type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage.
func New(store store.Store) Searcher {
	return &searcherImpl{
		store: store,
	}
}
