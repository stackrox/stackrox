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

// DeploymentTestStoreParams is a structure wrapping around the input
// parameters used to initialize a test datastore for Deployment objects.
type DeploymentTestStoreParams struct {
	ImagesDataStore                   imageDS.DataStore
	ProcessBaselinesDataStore         pbDS.DataStore
	NetworkGraphFlowClustersDataStore nfDS.ClusterDataStore
	RisksDataStore                    riskDS.DataStore
	DeletedDeploymentCache            expiringcache.Cache
	ProcessIndicatorFilter            filter.Filter
	ClusterRanker                     *ranking.Ranker
	NamespaceRanker                   *ranking.Ranker
	DeploymentRanker                  *ranking.Ranker
}

// NewTestDataStore allows for direct creation of the datastore for testing purposes
func NewTestDataStore(
	t testing.TB,
	testDB *pgtest.TestPostgres,
	storeParams *DeploymentTestStoreParams,
) (DataStore, error) {
	ctx := context.Background()
	pgStore.Destroy(ctx, testDB.DB)
	deploymentStore := pgStore.NewFullTestStore(ctx, t, pgStore.New(testDB.DB), testDB.GetGormDB(t))
	if t == nil {
		return nil, errors.New("NewTestDataStore called without testing")
	}

	searcher := search.NewV2(deploymentStore)
	ds := newDatastoreImpl(
		deploymentStore,
		searcher,
		storeParams.ImagesDataStore,
		storeParams.ProcessBaselinesDataStore,
		storeParams.NetworkGraphFlowClustersDataStore,
		storeParams.RisksDataStore,
		storeParams.DeletedDeploymentCache,
		storeParams.ProcessIndicatorFilter,
		storeParams.ClusterRanker,
		storeParams.NamespaceRanker,
		storeParams.DeploymentRanker,
	)

	ds.initializeRanker()
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	dbStore := pgStore.FullStoreWrap(pgStore.New(pool))
	searcher := search.NewV2(dbStore)
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
	return newDatastoreImpl(
		dbStore,
		searcher,
		imageStore,
		processBaselineStore,
		networkFlowClusterStore,
		riskStore,
		nil,
		processFilter,
		clusterRanker,
		namespaceRanker,
		deploymentRanker,
	), nil
}
