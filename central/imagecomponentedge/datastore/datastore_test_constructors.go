package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/globaldb"
	postgresStore "github.com/stackrox/rox/central/imagecomponentedge/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, error) {
	storage := postgresStore.New(pool)
	searcher := search.NewV2(storage, postgresStore.NewIndexer(globaldb.GetPostgres()))
	return New(storage, searcher)
}
