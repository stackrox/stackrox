package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/image/datastore/keyfence"
	"github.com/stackrox/rox/central/image/datastore/store"
	postgresStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	var dbstore store.Store
	if features.FlattenCVEData.Enabled() {
		dbstore = pgStoreV2.New(pool, false, keyfence.ImageKeyFenceSingleton())
	} else {
		dbstore = postgresStore.New(pool, false, keyfence.ImageKeyFenceSingleton())
	}
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return NewWithPostgres(dbstore, riskStore, imageRanker, imageComponentRanker)
}
