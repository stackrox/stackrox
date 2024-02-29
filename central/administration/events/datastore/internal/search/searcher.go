package search

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher provides search functionality on existing events.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given the store.
func New(store store.Store) Searcher {
	return &searcherImpl{
		store: store,
	}
}
