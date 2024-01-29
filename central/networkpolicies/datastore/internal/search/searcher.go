package search

import (
	"context"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Searcher provides search functionality on existing network policies
type Searcher interface {
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(index index.Indexer) Searcher {
	return &searcherImpl{
		index: index,
	}
}
