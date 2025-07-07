package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/discoveredclusters/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/discoveredclusters/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for modifying discovered cluster data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	CountDiscoveredClusters(ctx context.Context, query *v1.Query) (int, error)
	GetDiscoveredCluster(ctx context.Context, id string) (*storage.DiscoveredCluster, error)
	ListDiscoveredClusters(ctx context.Context, query *v1.Query) ([]*storage.DiscoveredCluster, error)
	UpsertDiscoveredClusters(ctx context.Context, discoveredClusters ...*discoveredclusters.DiscoveredCluster) error
	DeleteDiscoveredClusters(ctx context.Context, query *v1.Query) ([]string, error)
}

func newDataStore(storage store.Store) DataStore {
	return &datastoreImpl{
		store: storage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	return newDataStore(pgStore.New(pool))
}

// UpsertTestDiscoveredClusters provides a way to upsert storage.DiscoveredClusters directly to the database.
// This is required for testing with custom timestamps, since the datastore expects a struct with only a subset
// of fields that clients may set. We still want this to be the case for callers, however for testing we can
// be more lax in our enforcement.
func UpsertTestDiscoveredClusters(ctx context.Context, _ testing.TB, datastore DataStore,
	clusters ...*storage.DiscoveredCluster) error {
	return datastore.(*datastoreImpl).store.UpsertMany(ctx, clusters)
}
