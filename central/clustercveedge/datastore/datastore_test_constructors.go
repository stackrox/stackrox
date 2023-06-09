package datastore

import (
	"testing"

	pgStore "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/clustercveedge/search"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	storage := pgStore.NewFullStore(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.NewV2(storage, indexer)
	return New(nil, storage, indexer, searcher)
}
