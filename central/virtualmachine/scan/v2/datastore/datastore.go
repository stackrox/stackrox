package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to VirtualMachineScanV2 storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchRawVMScans(ctx context.Context, q *v1.Query) ([]*storage.VirtualMachineScanV2, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.VirtualMachineScanV2, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, ids []string) ([]*storage.VirtualMachineScanV2, error)
}

// New returns a new instance of a DataStore.
func New(storage pgStore.Store) DataStore {
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
