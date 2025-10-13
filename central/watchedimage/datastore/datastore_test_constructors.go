package datastore

import (
	"testing"

	postgresStore "github.com/stackrox/rox/central/watchedimage/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore returns a watched images datastore for testing
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	storage := postgresStore.New(pool)
	return New(storage)
}
