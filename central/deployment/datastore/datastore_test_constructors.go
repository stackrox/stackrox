package datastore

import (
	"errors"
	"testing"

	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/cache"
	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/deployment/datastore/internal/store/postgres"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	pbDS "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorFilter "github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"go.uber.org/mock/gomock"
)

// DeploymentTestStoreParams is a structure wrapping around the input
// parameters used to initialize a test datastore for Deployment objects.
type DeploymentTestStoreParams struct {
	ImagesDataStore                   imageDS.DataStore
	ProcessBaselinesDataStore         pbDS.DataStore
	NetworkGraphFlowClustersDataStore nfDS.ClusterDataStore
	RisksDataStore                    riskDS.DataStore
	DeletedDeploymentCache            cache.DeletedDeployments
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
	if t == nil {
		return nil, errors.New("NewTestDataStore called without testing")
	}
	deploymentStore := pgStore.FullStoreWrap(pgStore.New(testDB.DB))
	searcher := search.NewV2(deploymentStore)
	mockCtrl := gomock.NewController(t)
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(mockCtrl)
	mockConfigDatastore.EXPECT().GetPlatformComponentConfig(gomock.Any()).Return(&storage.PlatformComponentConfig{
		NeedsReevaluation: false,
		Rules: []*storage.PlatformComponentConfig_Rule{
			{
				Name: "system rule",
				NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
					Regex: `^kube-.*|^openshift-.*`,
				},
			},
			{
				Name: "red hat layered products",
				NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
					Regex: `^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`,
				},
			},
		},
	}, true, nil).Times(1)
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
		platformmatcher.New(mockConfigDatastore),
	)

	ds.initializeRanker()
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	dbStore := pgStore.FullStoreWrap(pgStore.New(pool))
	searcher := search.NewV2(dbStore)
	imageStore := imageDS.GetTestPostgresDataStore(t, pool)
	processBaselineStore := pbDS.GetTestPostgresDataStore(t, pool)
	networkFlowClusterStore, err := nfDS.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	processFilter := processIndicatorFilter.Singleton()
	clusterRanker := ranking.ClusterRanker()
	namespaceRanker := ranking.NamespaceRanker()
	deploymentRanker := ranking.DeploymentRanker()
	mockCtrl := gomock.NewController(t)
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(mockCtrl)
	mockConfigDatastore.EXPECT().GetPlatformComponentConfig(gomock.Any()).Return(&storage.PlatformComponentConfig{
		NeedsReevaluation: false,
		Rules: []*storage.PlatformComponentConfig_Rule{
			{
				Name: "system rule",
				NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
					Regex: `^kube-.*|^openshift-.*`,
				},
			},
			{
				Name: "red hat layered products",
				NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
					Regex: `^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$`,
				},
			},
		},
	}, true, nil).Times(1)
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
		platformmatcher.New(mockConfigDatastore),
	), nil
}
