package datastore

import (
	"testing"

	pgStore "github.com/stackrox/rox/central/hash/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) Datastore {
	return NewDatastore(pgStore.New(pool))
}
