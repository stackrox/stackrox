package search

import (
	"context"

	"github.com/stackrox/rox/central/discoveredclusters/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher provides search functionality on existing discovered clusters.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given indexer.
func New(store store.Store) Searcher {
	return &searcherImpl{
		store: store,
	}
}
