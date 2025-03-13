package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/nodecomponentedge/search"
	postgresStore "github.com/stackrox/rox/central/nodecomponentedge/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, error) {
	storage := postgresStore.New(pool)
	searcher := search.New(storage)
	return New(storage, searcher), nil
}
