package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/cve/cluster/datastore/search"
	pgStore "github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.NewFullStore(pool)
	searcher := search.New(dbstore, pgStore.NewIndexer(pool))
	return New(dbstore, searcher)
}
