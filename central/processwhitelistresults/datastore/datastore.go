package datastore

import (
	"context"

	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore wraps storage, indexer, and searcher for ProcessWhitelistResults.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	UpsertWhitelistResults(ctx context.Context, results *storage.ProcessWhitelistResults) error
	GetWhitelistResults(ctx context.Context, deploymentID string) (*storage.ProcessWhitelistResults, error)
	DeleteWhitelistResults(ctx context.Context, deploymentID string) error
}

// New returns a new instance of DataStore.
func New(storage store.Store) DataStore {
	d := &datastoreImpl{
		storage: storage,
	}
	return d
}
