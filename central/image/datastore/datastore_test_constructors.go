package datastore

import (
	"testing"

	postgresStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	dbstore := postgresStore.New(pool, false, concurrency.NewKeyFence())
	indexer := postgresStore.NewIndexer(pool)
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return NewWithPostgres(dbstore, indexer, riskStore, imageRanker, imageComponentRanker)
}
