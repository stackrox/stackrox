package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cloudsources/datastore/internal/search"
	"github.com/stackrox/rox/central/cloudsources/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/cloudsources/datastore/internal/store/postgres"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for modifying cloud sources data.
type DataStore interface {
	CountCloudSources(ctx context.Context, query *v1.Query) (int, error)
	GetCloudSource(ctx context.Context, id string) (*storage.CloudSource, error)
	ForEachCloudSource(ctx context.Context, fn func(obj *storage.CloudSource) error) error
	ListCloudSources(ctx context.Context, query *v1.Query) ([]*storage.CloudSource, error)
	UpsertCloudSource(ctx context.Context, cloudSource *storage.CloudSource) error
	DeleteCloudSource(ctx context.Context, id string) error
}

func newDataStore(searcher search.Searcher, storage store.Store,
	discoveredClusterDS discoveredClustersDS.DataStore) DataStore {
	return &datastoreImpl{
		searcher:            searcher,
		store:               storage,
		discoveredClusterDS: discoveredClusterDS,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	searcher := search.New(store)
	return newDataStore(searcher, store, discoveredClustersDS.GetTestPostgresDataStore(t, pool))
}
