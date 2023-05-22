package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store"
	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
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

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.New(pool)
	return New(dbstore), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, rocksengine *rocksdbBase.RocksDB) (DataStore, error) {
	dbstore := rocksdb.New(rocksengine)
	return New(dbstore), nil
}
