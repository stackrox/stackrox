package datastore

import (
	"context"
	"testing"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	pgStore "github.com/stackrox/rox/central/namespace/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
)

// NewTestDataStore returns a new DataStore instance.
func NewTestDataStore(t testing.TB, testDB *pgtest.TestPostgres, deploymentDataStore deploymentDataStore.DataStore, namespaceRanker *ranking.Ranker) DataStore {
	ctx := context.Background()
	pgStore.Destroy(ctx, testDB.DB)

	storage := pgStore.CreateTableAndNewStore(ctx, testDB.DB, testDB.GetGormDB(t))
	return New(storage, deploymentDataStore, namespaceRanker)
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	dbStore := pgStore.New(pool)
	deploymentStore, err := deploymentDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	namespaceRanker := ranking.NamespaceRanker()
	return New(dbStore, deploymentStore, namespaceRanker), nil
}
