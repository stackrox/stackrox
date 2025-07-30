package datastore

import (
	"testing"

	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	keyfenceV1 "github.com/stackrox/rox/central/image/datastore/keyfence"
	"github.com/stackrox/rox/central/image/datastore/store"
	postgresStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	keyfenceV2 "github.com/stackrox/rox/central/imagev2/datastore/keyfence"
	imageV2PgStore "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) imageDatastore.DataStore {
	var dbstore store.Store
	if features.FlattenCVEData.Enabled() {
		dbstore = pgStoreV2.New(pool, false, keyfenceV1.ImageKeyFenceSingleton())
	} else {
		dbstore = postgresStore.New(pool, false, keyfenceV1.ImageKeyFenceSingleton())
	}
	dbstore2 := imageV2PgStore.New(pool, false, keyfenceV2.ImageKeyFenceSingleton())
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return New(imageDatastore.NewWithPostgres(dbstore, riskStore, imageRanker, imageComponentRanker), imageV2Datastore.NewWithPostgres(dbstore2, riskStore, imageRanker, imageComponentRanker))
}
