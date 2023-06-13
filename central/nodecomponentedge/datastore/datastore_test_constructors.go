package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/nodecomponentedge/search"
	postgresStore "github.com/stackrox/rox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	storage := postgresStore.New(pool)
	indexer := postgresStore.NewIndexer(pool)
	searcher := search.New(storage, indexer)
	return New(storage, searcher), nil
}
