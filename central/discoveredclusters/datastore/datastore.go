package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/discoveredclusters/datastore/internal/search"
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

func newDataStore(searcher search.Searcher, storage store.Store) DataStore {
	return &datastoreImpl{
		searcher: searcher,
		store:    storage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	searcher := search.New(pgStore.NewIndexer(pool))
	store := pgStore.New(pool)
	return newDataStore(searcher, store)
}
