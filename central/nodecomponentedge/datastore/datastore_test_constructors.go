package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	dackboxIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/nodecomponentedge/search"
	dackboxStore "github.com/stackrox/rox/central/nodecomponentedge/store/dackbox"
	postgresStore "github.com/stackrox/rox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *postgres.DB) (DataStore, error) {
	storage := postgresStore.New(pool)
	indexer := postgresStore.NewIndexer(pool)
	searcher := search.New(storage, indexer)
	return New(nil, storage, indexer, searcher), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, bleveIndex bleve.Index, dacky *dackbox.DackBox) (DataStore, error) {
	storage := dackboxStore.New(dacky)
	indexer := dackboxIndex.New(bleveIndex)
	searcher := search.New(storage, indexer)
	return New(dacky, storage, indexer, searcher), nil
}
