package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	postgresStore "github.com/stackrox/rox/central/imagecomponentedge/datastore/internal/store/postgres"
	dackboxIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	dackboxStore "github.com/stackrox/rox/central/imagecomponentedge/store/dackbox"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *postgres.DB) (DataStore, error) {
	storage := postgresStore.New(pool)
	indexer := postgresStore.NewIndexer(pool)
	searcher := search.NewV2(storage, indexer)
	return New(nil, storage, indexer, searcher)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, bleveIndex bleve.Index, dacky *dackbox.DackBox) (DataStore, error) {
	storage, err := dackboxStore.New(dacky)
	if err != nil {
		return nil, err
	}
	indexer := dackboxIndex.New(bleveIndex)
	searcher := search.New(storage, indexer)
	return New(dacky, storage, indexer, searcher)
}
