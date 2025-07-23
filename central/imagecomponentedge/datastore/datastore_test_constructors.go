package datastore

import (
	"testing"

	postgresStore "github.com/stackrox/rox/central/imagecomponentedge/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	storage := postgresStore.New(pool)
	return New(storage)
}
