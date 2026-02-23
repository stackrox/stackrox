package store

import (
	"testing"

	pgStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) Store {
	dbStore := pgStore.New(pool)
	return NewStore(dbStore)
}
