package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/processbaselineresults/datastore/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
)

// DataStore wraps storage, indexer, and searcher for ProcessBaselineResults.
//go:generate mockgen-wrapper
type DataStore interface {
	UpsertBaselineResults(ctx context.Context, results *storage.ProcessBaselineResults) error
	GetBaselineResults(ctx context.Context, deploymentID string) (*storage.ProcessBaselineResults, error)
	DeleteBaselineResults(ctx context.Context, deploymentID string) error
}

// New returns a new instance of DataStore.
func New(storage store.Store) DataStore {
	d := &datastoreImpl{
		storage: storage,
	}
	return d
}
