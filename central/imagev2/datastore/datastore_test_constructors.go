package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/imagev2/datastore/keyfence"
	pgStore "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool, false, keyfence.ImageKeyFenceSingleton())
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return NewWithPostgres(dbstore, riskStore, imageRanker, imageComponentRanker)
}
