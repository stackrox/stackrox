package datastore

import (
	"testing"

	pgStore "github.com/stackrox/rox/central/virtualmachine/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	return newDatastoreImpl(pgStore.New(pool))
}
