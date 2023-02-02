package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v5/pgxpool"
	postgresStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgresStore.New(pool, false, concurrency.NewKeyFence())
	indexer := postgresStore.NewIndexer(pool)
	riskStore, err := riskDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return NewWithPostgres(dbstore, indexer, riskStore, imageRanker, imageComponentRanker), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (DataStore, error) {
	riskStore, err := riskDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return New(dacky, keyFence, bleveIndex, bleveIndex, false, riskStore, imageRanker, imageComponentRanker), nil
}
