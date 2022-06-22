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

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, t *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) DataStore {
	postgres.Destroy(ctx, pool)
	dbstore := postgres.CreateTableAndNewStore(ctx, pool, gormDB)
	return New(dbstore)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB) DataStore {
	dbstore := rocksdb.New(rocksengine)
	return New(dbstore)
}
