package datastore

import (
	"testing"

	pgStore "github.com/stackrox/rox/central/imagecveedge/datastore/postgres"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) DataStore {
	return New(pgStore.New(pool), search.NewV2(pgStore.New(pool),
		pgStore.NewIndexer(pool),
	))
}
