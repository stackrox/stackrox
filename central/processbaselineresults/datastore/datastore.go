package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore wraps storage, and searcher for ProcessBaselineResults.
//
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

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	return New(dbstore), nil
}
