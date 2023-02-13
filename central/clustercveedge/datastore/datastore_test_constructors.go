package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	pgStore "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/clustercveedge/index"
	"github.com/stackrox/rox/central/clustercveedge/search"
	dackboxStore "github.com/stackrox/rox/central/clustercveedge/store/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *postgres.DB) (DataStore, error) {
	storage := pgStore.NewFullStore(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.NewV2(storage, indexer)
	return New(nil, storage, indexer, searcher)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, _ *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence dackboxConcurrency.KeyFence) (DataStore, error) {
	storage, err := dackboxStore.New(dacky, keyFence)
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	cveIndexer := cveIndex.New(bleveIndex)
	searcher := search.New(storage, indexer, cveIndexer, dacky)
	return New(dacky, storage, indexer, searcher)
}
