package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cloudsources/datastore/internal/search"
	"github.com/stackrox/rox/central/cloudsources/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/cloudsources/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for modifying cloud sources data.
type DataStore interface {
	CountCloudSources(ctx context.Context, query *v1.Query) (int, error)
	GetCloudSource(ctx context.Context, id string) (*storage.CloudSource, error)
	ListCloudSources(ctx context.Context, query *v1.Query) ([]*storage.CloudSource, error)
	UpsertCloudSource(ctx context.Context, cloudSource *storage.CloudSource) error
	DeleteCloudSource(ctx context.Context, id string) error
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
