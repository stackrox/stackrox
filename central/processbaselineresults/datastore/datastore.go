package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store"
	postgresStore "github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/postgres"
	rocksdbStore "github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/rocksdb"
	"gorm.io/gorm"
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

// GetTestPostgresDataStore provides a processbaselineresult datastore hooked on rocksDB and bleve for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, _ *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) (DataStore, error) {
	dbstore := postgresStore.CreateTableAndNewStore(ctx, pool, gormDB)
	return New(dbstore), nil
}

// GetTestRocksBleveDataStore provides a processbaselineresult datastore hooked on rocksDB for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, rocksEngine *rocksdb.RocksDB) (DataStore, error) {
	dbstore := rocksdbStore.New(rocksEngine)
	return New(dbstore), nil
}
