package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/deployment/datastore/internal/store/postgres"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorFilter "github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
)

// NewTestDataStore allows for direct creation of the datastore for testing purposes
func NewTestDataStore(t testing.TB, testDB *pgtest.TestPostgres, images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore, risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter, clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	ctx := context.Background()
	pgStore.Destroy(ctx, testDB.DB)
	deploymentStore := pgStore.NewFullTestStore(ctx, t, pgStore.New(testDB.DB), testDB.GetGormDB(t))
	if t == nil {
		return nil, errors.New("NewTestDataStore called without testing")
	}

	searcher := search.NewV2(deploymentStore, pgStore.NewIndexer(testDB.DB))
	ds := newDatastoreImpl(deploymentStore, searcher, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)

	ds.initializeRanker()
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	dbStore := pgStore.FullStoreWrap(pgStore.New(pool))
	indexer := pgStore.NewIndexer(pool)
	searcher := search.NewV2(dbStore, indexer)
	imageStore := imageDS.GetTestPostgresDataStore(t, pool)
	processBaselineStore, err := pbDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkFlowClusterStore, err := nfDS.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	processFilter := processIndicatorFilter.Singleton()
	clusterRanker := ranking.ClusterRanker()
	namespaceRanker := ranking.NamespaceRanker()
	deploymentRanker := ranking.DeploymentRanker()
	return newDatastoreImpl(dbStore, searcher, imageStore, processBaselineStore, networkFlowClusterStore, riskStore, nil, processFilter, clusterRanker, namespaceRanker, deploymentRanker), nil
}
