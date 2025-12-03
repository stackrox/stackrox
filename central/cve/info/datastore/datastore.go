package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/info/datastore/store"
	pgStore "github.com/stackrox/rox/central/cve/info/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to the ImageCVETime storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchRawImageCVETimes(ctx context.Context, q *v1.Query) ([]*storage.ImageCVETime, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ImageCVETime, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ImageCVETime, error)
	Upsert(ctx context.Context, cve *storage.ImageCVETime) error
	UpsertMany(ctx context.Context, cve []*storage.ImageCVETime) error
}

// New returns a new instance of a DataStore.
func New(storage store.Store) DataStore {
	ds := &datastoreImpl{
		storage: storage,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	return New(dbstore)
}
