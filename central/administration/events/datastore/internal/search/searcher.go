package search

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher provides search functionality on existing notifications.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given indexer.
func New(indexer index.Indexer) Searcher {
	return &searcherImpl{
		indexer: indexer,
	}
}
