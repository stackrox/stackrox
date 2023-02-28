//go:build sql_integration

package pruning

import (
	"context"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	alertDatastoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterIndex "github.com/stackrox/rox/central/cluster/index"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterRocksDB "github.com/stackrox/rox/central/cluster/store/cluster/rocksdb"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterHealthRocksDB "github.com/stackrox/rox/central/cluster/store/clusterhealth/rocksdb"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageDatastoreMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentsMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	imageIntegrationDatastoreMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	logimbueDataStore "github.com/stackrox/rox/central/logimbue/store"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	netEntityMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	networkFlowDatastoreMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	testNodeDatastore "github.com/stackrox/rox/central/node/datastore"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	nodeSearch "github.com/stackrox/rox/central/node/datastore/search"
	nodePostgres "github.com/stackrox/rox/central/node/datastore/store/postgres"
	notifierMocks "github.com/stackrox/rox/central/notifier/processor/mocks"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	podMocks "github.com/stackrox/rox/central/pod/datastore/mocks"
	processBaselineDatastoreMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	plopDatastoreMocks "github.com/stackrox/rox/central/processlisteningonport/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	k8sRoleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	roleBindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	riskDatastore "github.com/stackrox/rox/central/risk/datastore"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	graphMocks "github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	filterMocks "github.com/stackrox/rox/pkg/process/filter/mocks"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	versionUtils "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testRetentionAttemptedDeploy  = 15
	testRetentionAttemptedRuntime = 15
	testRetentionResolvedDeploy   = 7
	testRetentionAllRuntime       = 6
	testRetentionDeletedRuntime   = 3
)

var (
	testConfig = &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention: &storage.PrivateConfig_AlertConfig{
				AlertConfig: &storage.AlertRetentionConfig{
					AllRuntimeRetentionDurationDays:       testRetentionAllRuntime,
					DeletedRuntimeRetentionDurationDays:   testRetentionDeletedRuntime,
					ResolvedDeployRetentionDurationDays:   testRetentionResolvedDeploy,
					AttemptedRuntimeRetentionDurationDays: testRetentionAttemptedRuntime,
					AttemptedDeployRetentionDurationDays:  testRetentionAttemptedDeploy,
				},
			},
			ImageRetentionDurationDays: configDatastore.DefaultImageRetention,
		},
	}
)

type PruningTestSuite struct {
	suite.Suite

	ctx  context.Context
	pool *postgres.DB
}

func (s *PruningTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

		testingDB := pgtest.ForT(s.T())
		s.pool = testingDB.DB
	}
}

func (s *PruningTestSuite) TearDownSuite() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.pool.Close()
	}
}

func TestPruning(t *testing.T) {
	suite.Run(t, new(PruningTestSuite))
}

func newAlertInstance(id string, daysOld int, stage storage.LifecycleStage, state storage.ViolationState) *storage.Alert {
	return newAlertInstanceWithDeployment(id, daysOld, stage, state, nil)
}
func newAlertInstanceWithDeployment(id string, daysOld int, stage storage.LifecycleStage, state storage.ViolationState, deployment *storage.Deployment) *storage.Alert {
	var alertDeployment *storage.Alert_Deployment_
	if deployment != nil {
		alertDeployment = convert.ToAlertDeployment(deployment)
	} else {
		alertDeployment = &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:       fixtureconsts.Deployment6,
				Inactive: true,
			},
		}
	}
	return &storage.Alert{
		Id: id,

		LifecycleStage: stage,
		State:          state,
		Entity:         alertDeployment,
		Time:           protoconv.ConvertTimeToTimestamp(time.Now().Add(-24 * time.Duration(daysOld) * time.Hour)),
	}
}

func newImageInstance(id string, daysOld int) *storage.Image {
	return &storage.Image{
		Id:          id,
		LastUpdated: protoconv.ConvertTimeToTimestamp(time.Now().Add(-24 * time.Duration(daysOld) * time.Hour)),
	}
}

func newDeployment(imageIDs ...string) *storage.Deployment {
	var containers []*storage.Container
	for _, id := range imageIDs {
		digest := types.NewDigest(id).Digest()
		containers = append(containers, &storage.Container{
			Image: &storage.ContainerImage{
				Id: digest,
			},
		})
	}
	return &storage.Deployment{
		Id:         fixtureconsts.Deployment1,
		Containers: containers,
	}
}

func newPod(live bool, imageIDs ...string) *storage.Pod {
	instanceLists := make([]*storage.Pod_ContainerInstanceList, len(imageIDs))
	instances := make([]*storage.ContainerInstance, len(imageIDs))
	for i, id := range imageIDs {
		if live {
			instances[i] = &storage.ContainerInstance{
				ImageDigest: types.NewDigest(id).Digest(),
			}
			// Populate terminated instances to ensure the indexing isn't overwritten.
			instanceLists[i] = &storage.Pod_ContainerInstanceList{
				Instances: []*storage.ContainerInstance{
					{
						ImageDigest: types.NewDigest("nonexistentid").Digest(),
					},
				},
			}
		} else {
			instanceLists[i] = &storage.Pod_ContainerInstanceList{
				Instances: []*storage.ContainerInstance{
					{
						ImageDigest: types.NewDigest(id).Digest(),
					},
				},
			}
		}
	}

	if live {
		return &storage.Pod{
			Id:                  fixtureconsts.PodUID1,
			LiveInstances:       instances,
			TerminatedInstances: instanceLists,
		}
	}

	return &storage.Pod{
		Id:                  fixtureconsts.PodUID2,
		TerminatedInstances: instanceLists,
	}
}

func setupRocksDBAndBleve(t *testing.T) (*rocksdb.RocksDB, bleve.Index) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return nil, nil
	}

	db := rocksdbtest.RocksDBForT(t)
	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	return db, bleveIndex
}

func (s *PruningTestSuite) generateImageDataStructures(ctx context.Context) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore, podDatastore.DataStore, queue.WaitableQueue) {
	// Setup the mocks
	ctrl := gomock.NewController(s.T())
	mockComponentDatastore := componentsMocks.NewMockDataStore(ctrl)
	mockComponentDatastore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes()
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockProcessDataStore := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)
	mockProcessDataStore.EXPECT().RemoveProcessIndicatorsByPod(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	mockBaselineDataStore := processBaselineDatastoreMocks.NewMockDataStore(ctrl)

	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)

	mockAlertDatastore := alertDatastoreMocks.NewMockDataStore(ctrl)

	mockFilter := filterMocks.NewMockFilter(ctrl)
	mockFilter.EXPECT().UpdateByPod(gomock.Any()).AnyTimes()
	mockFilter.EXPECT().DeleteByPod(gomock.Any()).AnyTimes()

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		db, bleveIndex := setupRocksDBAndBleve(s.T())

		dacky, registry, indexingQ := testDackBoxInstance(s.T(), db, bleveIndex)
		registry.RegisterWrapper(deploymentDackBox.Bucket, deploymentIndex.Wrapper{})
		registry.RegisterWrapper(imageDackBox.Bucket, imageIndex.Wrapper{})

		// Initialize real datastore
		images := imageDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), bleveIndex, bleveIndex, true, mockRiskDatastore, ranking.NewRanker(), ranking.NewRanker())

		deployments, err := deploymentDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, mockFilter, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
		require.NoError(s.T(), err)

		pods, err := podDatastore.NewRocksDB(db, bleveIndex, mockProcessDataStore, mockFilter)
		require.NoError(s.T(), err)

		return mockAlertDatastore, mockConfigDatastore, images, deployments, pods, indexingQ
	}

	deployments, err := deploymentDatastore.New(nil, dackboxConcurrency.NewKeyFence(), s.pool, nil, nil, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, mockFilter, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(s.T(), err)

	images := imageDatastore.NewWithPostgres(
		imagePostgres.New(s.pool, true, dackboxConcurrency.NewKeyFence()),
		imagePostgres.NewIndexer(s.pool),
		mockRiskDatastore,
		ranking.NewRanker(),
		ranking.NewRanker(),
	)

	pods, err := podDatastore.NewPostgresDB(s.pool, mockProcessDataStore, mockFilter)
	require.NoError(s.T(), err)

	return mockAlertDatastore, mockConfigDatastore, images, deployments, pods, nil
}

func (s *PruningTestSuite) generateNodeDataStructures() testNodeDatastore.DataStore {
	ctrl := gomock.NewController(s.T())
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		db, bleveIndex := setupRocksDBAndBleve(s.T())

		dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
		require.NoError(s.T(), err)

		nodes := testNodeDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), bleveIndex, mockRiskDatastore, ranking.NewRanker(), ranking.NewRanker())

		return nodes
	}

	nodeStore := nodePostgres.New(s.pool, false, dackboxConcurrency.NewKeyFence())
	nodeIndexer := nodePostgres.NewIndexer(s.pool)

	nodes := testNodeDatastore.NewWithPostgres(
		nodeStore,
		nodeIndexer,
		nodeSearch.NewV2(nodeStore, nodeIndexer),
		mockRiskDatastore,
		ranking.NewRanker(),
		ranking.NewRanker())

	return nodes
}

func (s *PruningTestSuite) generateAlertDataStructures(ctx context.Context) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db, bleveIndex := setupRocksDBAndBleve(s.T())

	// Initialize real datastore
	var (
		alerts alertDatastore.DataStore
		dacky  *dackbox.DackBox
		err    error
	)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		alerts, err = alertDatastore.GetTestPostgresDataStore(s.T(), s.pool)
		require.NoError(s.T(), err)
	} else {
		dacky, err = dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
		require.NoError(s.T(), err)

		alerts = alertDatastore.NewWithDb(db, bleveIndex)
	}

	ctrl := gomock.NewController(s.T())

	mockBaselineDataStore := processBaselineDatastoreMocks.NewMockDataStore(ctrl)

	mockImageDatastore := imageDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)

	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)

	deployments, err := deploymentDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), s.pool, bleveIndex, bleveIndex, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(s.T(), err)
	return alerts, mockConfigDatastore, mockImageDatastore, deployments
}

func (s *PruningTestSuite) generateClusterDataStructures() (configDatastore.DataStore, deploymentDatastore.DataStore, clusterDatastore.DataStore, queue.WaitableQueue) {
	// Setup mocks
	mockCtrl := gomock.NewController(s.T())
	mockBaselineDataStore := processBaselineDatastoreMocks.NewMockDataStore(mockCtrl)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	alertDataStore := alertDatastoreMocks.NewMockDataStore(mockCtrl)
	namespaceDataStore := namespaceMocks.NewMockDataStore(mockCtrl)
	nodeDataStore := nodeDatastoreMocks.NewMockDataStore(mockCtrl)
	podDataStore := podMocks.NewMockDataStore(mockCtrl)
	imageIntegrationDataStore := imageIntegrationDatastoreMocks.NewMockDataStore(mockCtrl)
	secretDataStore := secretMocks.NewMockDataStore(mockCtrl)
	flowsDataStore := networkFlowDatastoreMocks.NewMockClusterDataStore(mockCtrl)
	netEntityDataStore := netEntityMocks.NewMockEntityDataStore(mockCtrl)
	serviceAccountMockDataStore := serviceAccountMocks.NewMockDataStore(mockCtrl)
	roleDataStore := roleMocks.NewMockDataStore(mockCtrl)
	roleBindingDataStore := roleBindingMocks.NewMockDataStore(mockCtrl)
	connMgr := connectionMocks.NewMockManager(mockCtrl)
	notifierMock := notifierMocks.NewMockProcessor(mockCtrl)
	networkBaselineMgr := networkBaselineMocks.NewMockManager(mockCtrl)
	mockProvider := graphMocks.NewMockProvider(mockCtrl)
	mockFilter := filterMocks.NewMockFilter(mockCtrl)
	clusterFlows := networkFlowDatastoreMocks.NewMockClusterDataStore(mockCtrl)
	flows := networkFlowDatastoreMocks.NewMockFlowDataStore(mockCtrl)
	clusterCVEs := clusterCVEDS.NewMockDataStore(mockCtrl)

	// A bunch of these get called when a cluster is deleted
	flowsDataStore.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).AnyTimes().Return(networkFlowDatastoreMocks.NewMockFlowDataStore(mockCtrl), nil)
	connMgr.EXPECT().GetConnection(gomock.Any()).AnyTimes().Return(nil)
	namespaceDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]search.Result{}, nil)
	podDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	imageIntegrationDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	alertDataStore.EXPECT().SearchRawAlerts(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	notifierMock.EXPECT().ProcessAlert(gomock.Any(), gomock.Any()).AnyTimes().Return()
	podDataStore.EXPECT().RemovePod(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	imageIntegrationDataStore.EXPECT().RemoveImageIntegration(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	nodeDataStore.EXPECT().DeleteAllNodesForCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	serviceAccountMockDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	roleDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	roleBindingDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	netEntityDataStore.EXPECT().DeleteExternalNetworkEntitiesForCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	networkBaselineMgr.EXPECT().ProcessPostClusterDelete(gomock.Any()).AnyTimes().Return(nil)
	secretDataStore.EXPECT().RemoveSecret(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	serviceAccountMockDataStore.EXPECT().RemoveServiceAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	roleDataStore.EXPECT().RemoveRole(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	roleBindingDataStore.EXPECT().RemoveRoleBinding(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	mockRiskDatastore.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockBaselineDataStore.EXPECT().RemoveProcessBaselinesByDeployment(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	clusterFlows.EXPECT().GetFlowStore(gomock.Any(), gomock.Any()).AnyTimes().Return(flows, nil)
	flows.EXPECT().RemoveFlowsForDeployment(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	mockFilter.EXPECT().Delete(gomock.Any()).AnyTimes()
	clusterCVEs.EXPECT().DeleteClusterCVEsInternal(gomock.Any(), gomock.Any()).AnyTimes()

	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(mockCtrl)

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		db, bleveIndex := setupRocksDBAndBleve(s.T())
		clusterIndexer := clusterIndex.New(bleveIndex)

		dacky, registry, indexingQ := testDackBoxInstance(s.T(), db, bleveIndex)
		registry.RegisterWrapper(deploymentDackBox.Bucket, deploymentIndex.Wrapper{})

		deployments, err := deploymentDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, mockBaselineDataStore, clusterFlows,
			mockRiskDatastore, expiringcache.NewExpiringCache(1*time.Minute), mockFilter, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
		require.NoError(s.T(), err)

		clusterStorage, err := clusterRocksDB.New(db)
		require.NoError(s.T(), err)

		clusterHealthStorage, err := clusterHealthRocksDB.New(db)
		require.NoError(s.T(), err)

		nodeDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
		clusterDataStore, err := clusterDatastore.New(
			clusterStorage,
			clusterHealthStorage,
			clusterCVEs,
			alertDataStore,
			imageIntegrationDataStore,
			namespaceDataStore,
			deployments,
			nodeDataStore,
			podDataStore,
			secretDataStore,
			flowsDataStore,
			netEntityDataStore,
			serviceAccountMockDataStore,
			roleDataStore,
			roleBindingDataStore,
			connMgr,
			notifierMock,
			mockProvider,
			ranking.NewRanker(),
			clusterIndexer,
			networkBaselineMgr)
		require.NoError(s.T(), err)

		return mockConfigDatastore, deployments, clusterDataStore, indexingQ
	}

	deployments, err := deploymentDatastore.New(nil, dackboxConcurrency.NewKeyFence(), s.pool, nil, nil, nil, mockBaselineDataStore, clusterFlows, mockRiskDatastore, nil, mockFilter, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(s.T(), err)

	nodeDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	clusterDataStore, err := clusterDatastore.New(
		clusterPostgres.New(s.pool),
		clusterHealthPostgres.New(s.pool),
		clusterCVEs,
		alertDataStore,
		imageIntegrationDataStore,
		namespaceDataStore,
		deployments,
		nodeDataStore,
		podDataStore,
		secretDataStore,
		flowsDataStore,
		netEntityDataStore,
		serviceAccountMockDataStore,
		roleDataStore,
		roleBindingDataStore,
		connMgr,
		notifierMock,
		mockProvider,
		ranking.NewRanker(),
		clusterPostgres.NewIndexer(s.pool),
		networkBaselineMgr)
	require.NoError(s.T(), err)

	return mockConfigDatastore, deployments, clusterDataStore, nil
}

func (s *PruningTestSuite) TestImagePruning() {
	var cases = []struct {
		name        string
		images      []*storage.Image
		deployment  *storage.Deployment
		pod         *storage.Pod
		expectedIDs []string
	}{
		{
			name: "No pruning",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", 1),
			},
			expectedIDs: []string{"id1", "id2"},
		},
		{
			name: "one old and one new - no deployments nor pods",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 deployment with new",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id1"),
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 pod with new",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			pod:         newPod(true, "id1"),
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 pod with old",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			pod:         newPod(true, "id2"),
			expectedIDs: []string{"id1", "id2"},
		},
		{
			name: "two old - 1 deployment with old",
			images: []*storage.Image{
				newImageInstance("id1", configDatastore.DefaultImageRetention+1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id2"),
			expectedIDs: []string{"id2"},
		},
		{
			name: "two old - 1 deployment and pod with old",
			images: []*storage.Image{
				newImageInstance("id1", configDatastore.DefaultImageRetention+1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id2"),
			pod:         newPod(true, "id2"),
			expectedIDs: []string{"id2"},
		},
		{
			name: "two old - 1 pod with old",
			images: []*storage.Image{
				newImageInstance("id1", configDatastore.DefaultImageRetention+1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			pod:         newPod(true, "id2"),
			expectedIDs: []string{"id2"},
		},
		{
			name: "two old - 1 pod with old",
			images: []*storage.Image{
				newImageInstance("id1", configDatastore.DefaultImageRetention+1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			pod:         newPod(true, "id2"),
			expectedIDs: []string{"id2"},
		},
		{
			name: "two old - 1 deployment and pod with old, but have references to old",
			images: []*storage.Image{
				newImageInstance("id1", configDatastore.DefaultImageRetention+1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			deployment: &storage.Deployment{
				Id: fixtureconsts.Deployment1,
				Containers: []*storage.Container{
					{
						Image: &storage.ContainerImage{
							Id: "sha256:id1",
						},
					},
				},
			},
			pod:         newPod(true, "id2"),
			expectedIDs: []string{"id1", "id2"},
		},
		{
			name: "one new - 1 pod with new, but terminated",
			images: []*storage.Image{
				newImageInstance("id1", 1),
			},
			pod:         newPod(false, "id1"),
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old - 1 pod with old, but terminated",
			images: []*storage.Image{
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			pod:         newPod(false, "id2"),
			expectedIDs: []string{},
		},
	}

	scc := sac.TestScopeCheckerCoreFromAccessResourceMap(s.T(),
		[]permissions.ResourceWithAccess{
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Alert),
			// TODO: ROX-12750 Replace Config with Administration.
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Config),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Image),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.DeploymentExtension),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Alert),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Image),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.DeploymentExtension),
		})

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			alerts, config, images, deployments, pods, indexQ := s.generateImageDataStructures(ctx)
			nodes := s.generateNodeDataStructures()

			gc := newGarbageCollector(alerts, nodes, images, nil, deployments, pods, nil, nil, nil, nil, config, nil, nil, nil, nil, nil, nil, nil).(*garbageCollectorImpl)

			// Add images, deployments, and pods into the datastores
			if c.deployment != nil {
				require.NoError(t, deployments.UpsertDeployment(ctx, c.deployment))
			}
			if c.pod != nil {
				c.pod.DeploymentId = c.deployment.GetId()
				require.NoError(t, pods.UpsertPod(ctx, c.pod))
			}
			for _, image := range c.images {
				image.Id = types.NewDigest(image.Id).Digest()
				require.NoError(t, images.UpsertImage(ctx, image))
			}

			if !env.PostgresDatastoreEnabled.BooleanSetting() {
				indexingDone := concurrency.NewSignal()
				indexQ.PushSignal(&indexingDone)
				indexingDone.Wait()
			}

			conf, err := config.GetConfig(ctx)
			require.NoError(t, err, "failed to get config")
			// Garbage collect all of the images
			gc.collectImages(conf.GetPrivateConfig())

			// Grab the  actual remaining images and make sure they match the images expected to be remaining
			remainingImages, err := images.SearchListImages(ctx, search.EmptyQuery())
			require.NoError(t, err)

			var ids []string
			for _, i := range remainingImages {
				ids = append(ids, i.GetId())
			}
			for i, eid := range c.expectedIDs {
				c.expectedIDs[i] = types.NewDigest(eid).Digest()
			}

			assert.ElementsMatch(t, c.expectedIDs, ids)

			var cleanUpIDs []string
			for _, image := range c.images {
				cleanUpIDs = append(cleanUpIDs, image.Id)
			}
			require.NoError(t, images.DeleteImages(ctx, cleanUpIDs...))

			if c.pod != nil {
				require.NoError(t, pods.RemovePod(ctx, c.pod.Id))
			}
		})
	}
}

func (s *PruningTestSuite) TestClusterPruning() {
	s.T().Setenv(features.DecommissionedClusterRetention.EnvVar(), "true")
	if !features.DecommissionedClusterRetention.Enabled() {
		// if it's still not enabled, we're probably in release tests so skip
		s.T().Skip("Skipping because ROX_DECOMMISSIONED_CLUSTER_RETENTION feature flag isn't set.")
	}

	s.T().Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameRHACSRelease)

	versionUtils.SetExampleVersion(s.T())

	var cases = []struct {
		name          string
		recentlyRun   bool
		config        *storage.PrivateConfig
		clusters      []*storage.Cluster
		expectedNames []string
	}{
		{
			name:   "No pruning if config is set to 0 retention days",
			config: getCluserRetentionConfig(0, 90, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
				"UNHEALTHY cluster last contacted more than retention days ago",
			},
		},
		{
			name:        "No pruning if it hasn't been 24hrs since last run",
			recentlyRun: true,
			config:      getCluserRetentionConfig(60, 90, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
				"UNHEALTHY cluster last contacted more than retention days ago",
			},
		},
		{
			name:   "No pruning if config recently updated",
			config: getCluserRetentionConfig(60, 90, 23),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
				"UNHEALTHY cluster last contacted more than retention days ago",
			},
		},
		{
			name:   "No pruning if config was created less than retention days ago",
			config: getCluserRetentionConfig(10, 5, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster with last contact time before config creation time",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
				"UNHEALTHY cluster with last contact time before config creation time",
			},
		},
		{
			name:   "No pruning if only one cluster",
			config: getCluserRetentionConfig(60, 90, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"UNHEALTHY cluster last contacted more than retention days ago",
			},
		},
		{
			name:   "No pruning if all clusters are unhealthy",
			config: getCluserRetentionConfig(60, 90, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
				{
					Name:         "Another UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"UNHEALTHY cluster last contacted more than retention days ago",
				"Another UNHEALTHY cluster last contacted more than retention days ago",
			},
		},
		{
			name:   "Prune unhealthy cluster",
			config: getCluserRetentionConfig(6, 9, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
			},
		},
		{
			name:   "1 healthy cluster, 3 unhealthy clusters (1 excluded, 1 unhealthy recently, 1 past retention)",
			config: getCluserRetentionConfig(60, 90, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster matching a label to ignore the cluster",
					Labels:       map[string]string{"k2": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
				{
					Name:         "UNHEALTHY cluster with fewer than retentionDays since last contact",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(10),
				},
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
				"UNHEALTHY cluster matching a label to ignore the cluster",
				"UNHEALTHY cluster with fewer than retentionDays since last contact",
			},
		},
		{
			name:   "Multiple unhealthy clusters",
			config: getCluserRetentionConfig(60, 90, 72),
			clusters: []*storage.Cluster{
				{
					Name:         "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
				},
				{
					Name:         "UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(80),
				},
				{
					Name:         "Another UNHEALTHY cluster last contacted more than retention days ago",
					Labels:       map[string]string{"k1": "v2"},
					HealthStatus: unhealthyClusterStatus(100),
				},
			},
			expectedNames: []string{
				"HEALTHY cluster",
			},
		},
	}
	ctx := sac.WithAllAccess(context.Background())

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			_, deploymentsDS, clusterDS, _ := s.generateClusterDataStructures()

			for _, cluster := range c.clusters {
				clusterID, err := clusterDS.AddCluster(ctx, cluster)
				require.NoError(t, err)
				require.NoError(t, clusterDS.UpdateClusterHealth(ctx, clusterID, cluster.HealthStatus))
			}

			if c.recentlyRun {
				lastClusterPruneTime = time.Now()
			} else {
				lastClusterPruneTime = time.Now().Add(-24 * time.Hour)
			}

			gc := newGarbageCollector(nil, nil, nil, clusterDS, deploymentsDS, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).(*garbageCollectorImpl)
			gc.collectClusters(c.config)

			// Now get all clusters and compare the names to ensure only the expected ones exist
			finalClusters, err := clusterDS.GetClusters(ctx)
			require.NoError(t, err)
			require.Len(t, finalClusters, len(c.expectedNames), "Did not find expected number of clusters after gc")

			for _, cluster := range finalClusters {
				require.NotEqual(t, -1, sliceutils.Find(c.expectedNames, cluster.GetName()), "cluster %s should have been deleted", cluster.GetName())
				// Remove the cluster to be ready for next test
				require.NoError(t, clusterDS.RemoveCluster(s.ctx, cluster.GetId(), nil))
			}
		})
	}
}

func (s *PruningTestSuite) TestClusterPruningCentralCheck() {
	s.T().Setenv(features.DecommissionedClusterRetention.EnvVar(), "true")
	if !features.DecommissionedClusterRetention.Enabled() {
		// if it's still not enabled, we're probably in release tests so skip
		s.T().Skip("Skipping because ROX_DECOMMISSIONED_CLUSTER_RETENTION feature flag isn't set.")
	}

	s.T().Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameRHACSRelease)

	versionUtils.SetExampleVersion(s.T())

	var cases = []struct {
		name                string
		deploys             []*storage.Deployment
		shouldDeleteCluster bool
	}{
		{
			name:                "Don't prune if cluster has central",
			deploys:             []*storage.Deployment{customDeployment("central", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"})},
			shouldDeleteCluster: false,
		},
		{
			name:                "Don't prune if cluster has central in different namespace",
			deploys:             []*storage.Deployment{customDeployment("central", "myownnamespace", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"})},
			shouldDeleteCluster: false,
		},
		{
			name:                "Don't prune if cluster has central with extra labels",
			deploys:             []*storage.Deployment{customDeployment("central", "stackrox", map[string]string{"app": "central", "helm.sh/chart": "stackrox-central-services-70.0.0"}, map[string]string{"owner": "stackrox"})},
			shouldDeleteCluster: false,
		},
		{
			name:                "Don't prune if cluster has central with extra annotations",
			deploys:             []*storage.Deployment{customDeployment("central", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox", "meta.helm.sh/release-name": "stackrox-central-services"})},
			shouldDeleteCluster: false,
		},
		{
			name:                "Prune if cluster has non-central deployment based on name",
			deploys:             []*storage.Deployment{customDeployment("centrally", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"})},
			shouldDeleteCluster: true,
		},
		{
			name:                "Prune if cluster has non-central deployment based on label",
			deploys:             []*storage.Deployment{customDeployment("central", "stackrox", map[string]string{"app": "centrally"}, map[string]string{"owner": "stackrox"})},
			shouldDeleteCluster: true,
		},
		{
			name:                "Prune if cluster has non-central deployment based on annotation",
			deploys:             []*storage.Deployment{customDeployment("central", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackroxy"})},
			shouldDeleteCluster: true,
		},
		{
			name: "Don't prune if cluster has multiple centrals",
			deploys: []*storage.Deployment{
				customDeployment("central", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"}),
				customDeployment("central", "stackrox2", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"}),
			},
			shouldDeleteCluster: false,
		},
		{
			name: "Don't prune if cluster has multiple deploys with one being central",
			deploys: []*storage.Deployment{
				customDeployment("central", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"}),
				customDeployment("centrally", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"}),
			},
			shouldDeleteCluster: false,
		},
		{
			name: "Prune if cluster has multiple deploys with none being central",
			deploys: []*storage.Deployment{
				customDeployment("central", "stackrox", map[string]string{"app": "centrally"}, map[string]string{"owner": "stackrox"}),
				customDeployment("centrally", "stackrox", map[string]string{"app": "central"}, map[string]string{"owner": "stackrox"}),
			},
			shouldDeleteCluster: true,
		},
	}
	ctx := sac.WithAllAccess(context.Background())

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			_, deploymentsDS, clusterDS, indexingQ := s.generateClusterDataStructures()

			// Add the unhealthy cluster that is under test
			cluster := &storage.Cluster{
				Name:         "Unhealthy cluster",
				HealthStatus: unhealthyClusterStatus(80),
			}
			clusterID, err := clusterDS.AddCluster(ctx, cluster)
			require.NoError(t, err)
			require.NoError(t, clusterDS.UpdateClusterHealth(ctx, clusterID, cluster.HealthStatus))

			// Add the deployments whose params are being changed for this test
			for _, d := range c.deploys {
				d.ClusterId = cluster.GetId()
				d.ClusterName = cluster.GetName()
				require.NoError(t, deploymentsDS.UpsertDeployment(ctx, d))
			}

			// Add another random deployment in just for variety
			randDeploy := fixtures.GetDeployment()
			randDeploy.ClusterId = cluster.GetId()
			require.NoError(t, deploymentsDS.UpsertDeployment(ctx, randDeploy))

			// Add in a healthy cluster because GC won't run unless there are two cluster
			_, err = clusterDS.AddCluster(ctx, &storage.Cluster{
				Name:         "HEALTHY cluster",
				HealthStatus: &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY},
			})
			require.NoError(t, err)

			if !env.PostgresDatastoreEnabled.BooleanSetting() {
				// Wait for deployments to get indexed before continuing
				newSig := concurrency.NewSignal()
				indexingQ.PushSignal(&newSig)
				newSig.Wait()
			}

			// Run GC
			lastClusterPruneTime = time.Now().Add(-24 * time.Hour)
			gc := newGarbageCollector(nil, nil, nil, clusterDS, deploymentsDS, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).(*garbageCollectorImpl)
			gc.collectClusters(getCluserRetentionConfig(60, 90, 72))

			// Now get all clusters and compare the names to ensure only the expected ones exist
			finalClusters, err := clusterDS.GetClusters(ctx)
			require.NoError(t, err)

			expectedClusters := map[string]bool{
				cluster.GetName(): !c.shouldDeleteCluster,
				"HEALTHY cluster": true,
			}

			for _, cluster := range finalClusters {
				require.True(t, expectedClusters[cluster.GetName()], "cluster %s should have been deleted", cluster.GetName())
				// Remove the cluster to be ready for next test
				require.NoError(t, clusterDS.RemoveCluster(s.ctx, cluster.GetId(), nil))
			}
		})
	}
}

func unhealthyClusterStatus(daysSinceLastContact int) *storage.ClusterHealthStatus {
	return &storage.ClusterHealthStatus{
		SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
		LastContact:        timeBeforeDays(daysSinceLastContact),
	}
}

func getCluserRetentionConfig(retentionDays int, createdBeforeDays int, lastUpdatedBeforeHours int) *storage.PrivateConfig {
	return &storage.PrivateConfig{
		DecommissionedClusterRetention: &storage.DecommissionedClusterRetentionConfig{
			RetentionDurationDays: int32(retentionDays),
			IgnoreClusterLabels: map[string]string{
				"k1": "v1",
				"k2": "v2",
				"k3": "v3",
			},
			LastUpdated: timeBeforeHours(lastUpdatedBeforeHours),
			CreatedAt:   timeBeforeDays(createdBeforeDays),
		}}
}

func customDeployment(name string, namespace string, labels map[string]string, annotations map[string]string) *storage.Deployment {
	deploy := fixtures.LightweightDeployment()
	deploy.Id = uuid.NewV4().String()
	deploy.Name = name
	deploy.Namespace = namespace
	deploy.Labels = labels
	deploy.Annotations = annotations
	return deploy
}

func (s *PruningTestSuite) TestAlertPruning() {
	existsDeployment := &storage.Deployment{
		Id:        fixtureconsts.Deployment1,
		Name:      "test deployment",
		Namespace: "ns",
		ClusterId: fixtureconsts.Cluster1,
	}

	var cases = []struct {
		name                 string
		alerts               []*storage.Alert
		expectedIDsRemaining []string
		deployments          []*storage.Deployment
	}{
		{
			name: "No pruning",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert1, 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
				newAlertInstance(fixtureconsts.Alert2, 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
				newAlertInstance(fixtureconsts.Alert3, 1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ATTEMPTED),
				newAlertInstance(fixtureconsts.Alert4, 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3, fixtureconsts.Alert4},
		},
		{
			name: "One old alert, and one new alert",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert1, 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
				newAlertInstance(fixtureconsts.Alert2, testRetentionAllRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert1},
		},
		{
			name: "One old runtime alert, and one old deploy time unresolved alert",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert1, testRetentionAllRuntime+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newAlertInstance(fixtureconsts.Alert2, testRetentionAllRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert1},
		},
		{
			name: "one old deploy time alert resolved",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert1, testRetentionResolvedDeploy+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED),
			},
			expectedIDsRemaining: []string{},
		},
		{
			name: "two old-ish runtime alerts, one with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment(fixtureconsts.Alert1, testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED, nil),
				newAlertInstanceWithDeployment(fixtureconsts.Alert2, testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED, existsDeployment),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert2},
			deployments: []*storage.Deployment{
				existsDeployment,
			},
		},
		{
			name: "expired runtime alert with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment(fixtureconsts.Alert1, testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE, nil),
			},
			expectedIDsRemaining: []string{},
		},
		{
			name: "One old attempted deploy alert, and one new attempted deploy alert",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert1, 1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ATTEMPTED),
				newAlertInstance(fixtureconsts.Alert2, testRetentionAttemptedDeploy+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert1},
		},
		{
			name: "Attempted runtime retention > deleted runtime retention",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert1, 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
				newAlertInstance(fixtureconsts.Alert2, testRetentionDeletedRuntime-1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
				newAlertInstance(fixtureconsts.Alert3, testRetentionAttemptedRuntime-1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert1, fixtureconsts.Alert2},
		},
		{
			name: "Attempted runtime alert with no deployment",
			alerts: []*storage.Alert{
				newAlertInstance(fixtureconsts.Alert2, testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{},
		},
		{
			name: "Attempted runtime alerts, one with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment(fixtureconsts.Alert1, testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED, nil),
				newAlertInstanceWithDeployment(fixtureconsts.Alert2, testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED, existsDeployment),
			},
			expectedIDsRemaining: []string{fixtureconsts.Alert2},
			deployments: []*storage.Deployment{
				existsDeployment,
			},
		},
	}
	scc := sac.TestScopeCheckerCoreFromAccessResourceMap(s.T(),
		[]permissions.ResourceWithAccess{
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Alert),
			// TODO: ROX-12750 Replace Config with Administration.
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Config),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Image),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Alert),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Image),
		})

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			alerts, config, images, deployments := s.generateAlertDataStructures(ctx)
			nodes := s.generateNodeDataStructures()

			gc := newGarbageCollector(alerts, nodes, images, nil, deployments, nil, nil, nil, nil, nil, config, nil, nil, nil, nil, nil, nil, nil).(*garbageCollectorImpl)

			// Add alerts into the datastores
			for _, alert := range c.alerts {
				require.NoError(t, alerts.UpsertAlert(ctx, alert))
			}
			for _, deployment := range c.deployments {
				require.NoError(t, deployments.UpsertDeployment(ctx, deployment))
			}
			all, err := alerts.Search(ctx, getAllAlerts())
			if err != nil {
				t.Error(err)
			}
			log.Infof("All query returns %d objects: %v", len(all), search.ResultsToIDs(all))

			conf, err := config.GetConfig(ctx)
			require.NoError(t, err, "failed to get config")

			// Garbage collect all of the alerts
			gc.collectAlerts(conf.GetPrivateConfig())

			// Grab the actual remaining alerts and make sure they match the alerts expected to be remaining
			remainingAlerts, err := alerts.SearchListAlerts(ctx, getAllAlerts())
			require.NoError(t, err)

			log.Infof("Remaining alerts: %v", remainingAlerts)
			var ids []string
			for _, i := range remainingAlerts {
				ids = append(ids, i.GetId())
			}

			assert.ElementsMatch(t, c.expectedIDsRemaining, ids)

			// Clear out the remaining alerts for the next run
			err = alerts.DeleteAlerts(s.ctx, ids...)
			require.NoError(t, err)
		})
	}
}

func timeBeforeDays(days int) *protoTypes.Timestamp {
	return timestampNowMinus(24 * time.Duration(days) * time.Hour)
}

func timeBeforeHours(hours int) *protoTypes.Timestamp {
	return timestampNowMinus(time.Duration(hours) * time.Hour)
}

func newListAlertWithDeployment(id string, age time.Duration, deploymentID string, stage storage.LifecycleStage, state storage.ViolationState) *storage.ListAlert {
	return &storage.ListAlert{
		Id: id,
		Entity: &storage.ListAlert_Deployment{
			Deployment: &storage.ListAlertDeployment{Id: deploymentID},
		},
		State:          state,
		LifecycleStage: stage,
		Time:           timestampNowMinus(age),
	}
}

func newIndicatorWithDeployment(id string, age time.Duration, deploymentID string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            id,
		DeploymentId:  deploymentID,
		ContainerName: "",
		PodId:         "",
		Signal: &storage.ProcessSignal{
			Time: timestampNowMinus(age),
		},
	}
}

func newIndicatorWithDeploymentAndPod(id string, age time.Duration, deploymentID, podUID string) *storage.ProcessIndicator {
	indicator := newIndicatorWithDeployment(id, age, deploymentID)
	indicator.PodUid = podUID
	return indicator
}

func (s *PruningTestSuite) TestRemoveOrphanedProcesses() {
	cases := []struct {
		name              string
		initialProcesses  []*storage.ProcessIndicator
		deployments       set.FrozenStringSet
		pods              set.FrozenStringSet
		expectedDeletions []string
	}{
		{
			name: "no deployments nor pods - remove all old indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment2, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(),
			pods:              set.NewFrozenStringSet(),
			expectedDeletions: []string{fixtureconsts.ProcessIndicatorID1, fixtureconsts.ProcessIndicatorID2, fixtureconsts.ProcessIndicatorID3},
		},
		{
			name: "no deployments nor pods - remove no new orphaned indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 20*time.Minute, fixtureconsts.Deployment1, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment2, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 20*time.Minute, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(),
			pods:              set.NewFrozenStringSet(),
			expectedDeletions: []string{},
		},
		{
			name: "all pods separate deployments - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment2, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment1, fixtureconsts.Deployment2, fixtureconsts.Deployment3),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedDeletions: []string{},
		},
		{
			name: "all pods same deployment - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment1),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedDeletions: []string{},
		},
		{
			name: "some pods separate deployments - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment2, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment3),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID3),
			expectedDeletions: []string{fixtureconsts.ProcessIndicatorID1},
		},
		{
			name: "some pods same deployment - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment1, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment1, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment1),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID3),
			expectedDeletions: []string{fixtureconsts.ProcessIndicatorID1},
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			processes := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)
			gci := &garbageCollectorImpl{
				processes: processes,
			}

			processes.EXPECT().WalkAll(pruningCtx, gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(pi *storage.ProcessIndicator) error) error {
					for _, a := range c.initialProcesses {
						assert.NoError(t, fn(a))
					}
					return nil
				})
			processes.EXPECT().RemoveProcessIndicators(pruningCtx, testutils.AssertionMatcher(assert.ElementsMatch, c.expectedDeletions))
			gci.removeOrphanedProcesses(c.deployments, c.pods)
		})
	}
}

func (s *PruningTestSuite) TestRemoveOrphanedPLOPs() {
	plopID1 := "asdf"

	cases := []struct {
		name              string
		initialPlops      []*storage.ProcessListeningOnPortStorage
		expectedDeletions []string
	}{
		{
			name: "Plop is active so it should not be removed",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				{
					Id:                 plopID1,
					Port:               1234,
					Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
					CloseTimestamp:     nil,
					ProcessIndicatorId: fixtureconsts.ProcessIndicatorID1,
					Closed:             false,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               fixtureconsts.PodUID1,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: "test_path1",
					},
				},
			},
			expectedDeletions: []string{},
		},
		{
			name: "Plop is closed but not expired so it is not removed",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				{
					Id:                 plopID1,
					Port:               1234,
					Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
					CloseTimestamp:     timestampNowMinus(1 * time.Second),
					ProcessIndicatorId: fixtureconsts.ProcessIndicatorID1,
					Closed:             true,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               fixtureconsts.PodUID1,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: "test_path1",
					},
				},
			},
			expectedDeletions: []string{},
		},
		{
			name: "Plop is expired so it should be removed",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				{
					Id:                 plopID1,
					Port:               1234,
					Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
					CloseTimestamp:     timestampNowMinus(1 * time.Hour),
					ProcessIndicatorId: fixtureconsts.ProcessIndicatorID1,
					Closed:             true,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               fixtureconsts.PodUID1,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: "test_path1",
					},
				},
			},
			expectedDeletions: []string{plopID1},
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			plops := plopDatastoreMocks.NewMockDataStore(ctrl)
			gci := &garbageCollectorImpl{
				plops: plops,
			}

			plops.EXPECT().WalkAll(pruningCtx, gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(pi *storage.ProcessListeningOnPortStorage) error) error {
					for _, a := range c.initialPlops {
						assert.NoError(t, fn(a))
					}
					return nil
				})
			plops.EXPECT().RemoveProcessListeningOnPort(pruningCtx, testutils.AssertionMatcher(assert.ElementsMatch, c.expectedDeletions))
			gci.removeOrphanedPLOP()
		})
	}
}

func (s *PruningTestSuite) TestMarkOrphanedAlerts() {
	cases := []struct {
		name              string
		initialAlerts     []*storage.ListAlert
		deployments       set.FrozenStringSet
		expectedDeletions []string
	}{
		{
			name: "no deployments - remove all old alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment(fixtureconsts.Alert1, 1*time.Hour, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert2, 1*time.Hour, fixtureconsts.Deployment2, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(),
			expectedDeletions: []string{fixtureconsts.Alert1, fixtureconsts.Alert2},
		},
		{
			name: "no deployments - remove no new orphaned alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment(fixtureconsts.Alert1, 20*time.Minute, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert2, 20*time.Minute, fixtureconsts.Deployment2, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert3, 20*time.Minute, fixtureconsts.Deployment3, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(),
			expectedDeletions: []string{},
		},
		{
			name: "all deployments - remove no alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment(fixtureconsts.Alert1, 1*time.Hour, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert2, 1*time.Hour, fixtureconsts.Deployment2, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert3, 1*time.Hour, fixtureconsts.Deployment3, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment1, fixtureconsts.Deployment2, fixtureconsts.Deployment3),
			expectedDeletions: []string{},
		},
		{
			name: "some deployments - remove some alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment(fixtureconsts.Alert1, 1*time.Hour, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert2, 20*time.Minute, fixtureconsts.Deployment2, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert3, 1*time.Hour, fixtureconsts.Deployment3, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment3),
			expectedDeletions: []string{fixtureconsts.Alert1},
		},
		{
			name: "some deployments - remove some alerts due to stages",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment(fixtureconsts.Alert1, 1*time.Hour, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert2, 1*time.Hour, fixtureconsts.Deployment2, storage.LifecycleStage_BUILD, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert3, 1*time.Hour, fixtureconsts.Deployment3, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment3),
			expectedDeletions: []string{fixtureconsts.Alert1},
		},
		{
			name: "some deployments - remove some alerts due to state",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment(fixtureconsts.Alert1, 1*time.Hour, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment(fixtureconsts.Alert2, 1*time.Hour, fixtureconsts.Deployment2, storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED),
				newListAlertWithDeployment(fixtureconsts.Alert3, 1*time.Hour, fixtureconsts.Deployment3, storage.LifecycleStage_DEPLOY, storage.ViolationState_SNOOZED),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment3),
			expectedDeletions: []string{fixtureconsts.Alert1},
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			alerts := alertDatastoreMocks.NewMockDataStore(ctrl)
			gci := &garbageCollectorImpl{
				alerts: alerts,
			}
			alerts.EXPECT().WalkAll(pruningCtx, gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(la *storage.ListAlert) error) error {
					for _, a := range c.initialAlerts {
						assert.NoError(t, fn(a))
					}
					return nil
				})
			alerts.EXPECT().MarkAlertStaleBatch(pruningCtx, c.expectedDeletions)
			gci.markOrphanedAlertsAsResolved(c.deployments)
		})
	}
}

func (s *PruningTestSuite) TestRemoveOrphanedNetworkFlows() {
	cases := []struct {
		name             string
		flows            []*storage.NetworkFlow
		deployments      set.FrozenStringSet
		expectedDeletion bool
	}{
		{
			name: "no deployments - remove all flows",
			flows: []*storage.NetworkFlow{
				{
					LastSeenTimestamp: timestampNowMinus(1 * time.Hour),
					Props: &storage.NetworkFlowProperties{
						SrcEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment1,
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment2,
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet(),
			expectedDeletion: true,
		},
		{
			name: "no deployments - but no flows with deployments",
			flows: []*storage.NetworkFlow{
				{
					LastSeenTimestamp: timestampNowMinus(1 * time.Hour),
					Props: &storage.NetworkFlowProperties{
						SrcEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_INTERNET,
							Id:   "i1",
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_INTERNET,
							Id:   "i2",
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet(),
			expectedDeletion: false,
		},
		{
			name: "no deployments - but flows too recent",
			flows: []*storage.NetworkFlow{
				{
					LastSeenTimestamp: timestampNowMinus(20 * time.Minute),
					Props: &storage.NetworkFlowProperties{
						SrcEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment1,
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment2,
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet(),
			expectedDeletion: false,
		},
		{
			name: "some deployments with matching flows",
			flows: []*storage.NetworkFlow{
				{
					LastSeenTimestamp: timestampNowMinus(1 * time.Hour),
					Props: &storage.NetworkFlowProperties{
						SrcEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment1,
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment2,
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet(fixtureconsts.Deployment1, fixtureconsts.Deployment2),
			expectedDeletion: false,
		},
		{
			name: "some deployments with matching src",
			flows: []*storage.NetworkFlow{
				{
					LastSeenTimestamp: timestampNowMinus(1 * time.Hour),
					Props: &storage.NetworkFlowProperties{
						SrcEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment1,
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment2,
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet(fixtureconsts.Deployment1),
			expectedDeletion: true,
		},
		{
			name: "some deployments with matching dst",
			flows: []*storage.NetworkFlow{
				{
					LastSeenTimestamp: timestampNowMinus(1 * time.Hour),
					Props: &storage.NetworkFlowProperties{
						SrcEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment1,
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   fixtureconsts.Deployment2,
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet(fixtureconsts.Deployment2),
			expectedDeletion: true,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			clusterFlows := networkFlowDatastoreMocks.NewMockClusterDataStore(ctrl)
			flows := networkFlowDatastoreMocks.NewMockFlowDataStore(ctrl)

			clusterFlows.EXPECT().GetFlowStore(pruningCtx, fixtureconsts.Cluster1).Return(flows, nil)

			if env.PostgresDatastoreEnabled.BooleanSetting() {
				flows.EXPECT().RemoveStaleFlows(pruningCtx).Return(nil)
			}
			flows.EXPECT().RemoveMatchingFlows(pruningCtx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, keyFn func(props *storage.NetworkFlowProperties) bool, valueFn func(flow *storage.NetworkFlow) bool) error {
					var deleted bool
					for _, f := range c.flows {
						if !keyFn(f.Props) || !valueFn(f) {
							continue
						}
						deleted = true
					}
					assert.Equal(t, c.expectedDeletion, deleted)
					return nil
				})

			gci := &garbageCollectorImpl{
				networkflows: clusterFlows,
			}
			gci.removeOrphanedNetworkFlows(c.deployments, set.NewFrozenStringSet(fixtureconsts.Cluster1))
		})
	}
}

func (s *PruningTestSuite) TestRemoveOrphanedImageRisks() {
	id1, _ := riskDatastore.GetID("img1", storage.RiskSubjectType_IMAGE)
	id2, _ := riskDatastore.GetID("img2", storage.RiskSubjectType_IMAGE)
	id3, _ := riskDatastore.GetID("img3", storage.RiskSubjectType_IMAGE)
	id4, _ := riskDatastore.GetID("img4", storage.RiskSubjectType_IMAGE)

	cases := []struct {
		name              string
		risks             []search.Result
		images            []search.Result
		expectedDeletions []string
	}{
		{
			name: "no images - remove all risk",
			risks: []search.Result{
				{ID: id1},
				{ID: id2},
			},
			images:            []search.Result{},
			expectedDeletions: []string{"img1", "img2"},
		},
		{
			name: "all images - remove no orphaned risk",
			risks: []search.Result{
				{ID: id1},
				{ID: id2},
				{ID: id3},
			},
			images: []search.Result{
				{ID: "img1"},
				{ID: "img2"},
				{ID: "img3"},
			},
			expectedDeletions: []string{},
		},
		{
			name: "some images - remove some risk",
			risks: []search.Result{
				{ID: id1},
				{ID: id2},
				{ID: id3},
				{ID: id4},
			},
			images: []search.Result{
				{ID: "img1"},
			},
			expectedDeletions: []string{"img2", "img3", "img4"},
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			images := imageDatastoreMocks.NewMockDataStore(ctrl)
			risks := riskDatastoreMocks.NewMockDataStore(ctrl)
			gci := &garbageCollectorImpl{
				images: images,
				risks:  risks,
			}

			risks.EXPECT().Search(gomock.Any(), gomock.Any()).Return(c.risks, nil)
			images.EXPECT().Search(gomock.Any(), gomock.Any()).Return(c.images, nil)
			for _, id := range c.expectedDeletions {
				risks.EXPECT().RemoveRisk(gomock.Any(), id, storage.RiskSubjectType_IMAGE).Return(nil)
			}

			gci.removeOrphanedImageRisks()
		})
	}
}

func (s *PruningTestSuite) TestRemoveOrphanedNodeRisks() {
	nodeID1, _ := riskDatastore.GetID("node1", storage.RiskSubjectType_NODE)
	nodeID2, _ := riskDatastore.GetID("node2", storage.RiskSubjectType_NODE)
	nodeID3, _ := riskDatastore.GetID("node3", storage.RiskSubjectType_NODE)
	nodeID4, _ := riskDatastore.GetID("node4", storage.RiskSubjectType_NODE)

	cases := []struct {
		name              string
		risks             []search.Result
		nodes             []search.Result
		expectedDeletions []string
	}{
		{
			name: "no nodes - remove all risk",
			risks: []search.Result{
				{ID: nodeID1},
				{ID: nodeID2},
			},
			nodes:             []search.Result{},
			expectedDeletions: []string{"node1", "node2"},
		},
		{
			name: "all nodes - remove no orphaned risk",
			risks: []search.Result{
				{ID: nodeID1},
				{ID: nodeID2},
				{ID: nodeID3},
			},
			nodes: []search.Result{
				{ID: "node1"},
				{ID: "node2"},
				{ID: "node3"},
			},
			expectedDeletions: []string{},
		},
		{
			name: "some nodes - remove some risk",
			risks: []search.Result{
				{ID: nodeID1},
				{ID: nodeID2},
				{ID: nodeID3},
				{ID: nodeID4},
			},
			nodes: []search.Result{
				{ID: "node1"},
			},
			expectedDeletions: []string{"node2", "node3", "node4"},
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nodes := nodeDatastoreMocks.NewMockDataStore(ctrl)
			risks := riskDatastoreMocks.NewMockDataStore(ctrl)
			gci := &garbageCollectorImpl{
				nodes: nodes,
				risks: risks,
			}

			risks.EXPECT().Search(gomock.Any(), gomock.Any()).Return(c.risks, nil)
			nodes.EXPECT().Search(gomock.Any(), gomock.Any()).Return(c.nodes, nil)
			for _, id := range c.expectedDeletions {
				risks.EXPECT().RemoveRisk(gomock.Any(), id, storage.RiskSubjectType_NODE).Return(nil)
			}

			gci.removeOrphanedNodeRisks()
		})
	}
}

func (s *PruningTestSuite) TestRemoveOrphanedRBACObjects() {
	clusters := []string{uuid.NewV4().String(), uuid.NewV4().String(), uuid.NewV4().String()}
	cases := []struct {
		name                  string
		validClusters         []string
		serviceAccts          []*storage.ServiceAccount
		roles                 []*storage.K8SRole
		bindings              []*storage.K8SRoleBinding
		expectedSADeletions   set.FrozenStringSet
		expectedRoleDeletions set.FrozenStringSet
		expectedRBDeletions   set.FrozenStringSet
	}{
		{
			name:          "remove SAs that belong to deleted clusters",
			validClusters: clusters,
			serviceAccts: []*storage.ServiceAccount{
				{Id: fixtureconsts.ServiceAccount1, ClusterId: clusters[0]},
				{Id: fixtureconsts.ServiceAccount2, ClusterId: fixtureconsts.ClusterFake1},
				{Id: fixtureconsts.ServiceAccount3, ClusterId: clusters[1]},
				{Id: fixtureconsts.ServiceAccount4, ClusterId: fixtureconsts.ClusterFake2},
			},
			expectedSADeletions: set.NewFrozenStringSet(fixtureconsts.ServiceAccount2, fixtureconsts.ServiceAccount4),
		},
		{
			name:          "Removing when there is only one valid cluster",
			validClusters: clusters[:1],
			serviceAccts: []*storage.ServiceAccount{
				{Id: fixtureconsts.ServiceAccount1, ClusterId: clusters[0]},
				{Id: fixtureconsts.ServiceAccount2, ClusterId: fixtureconsts.ClusterFake1},
				{Id: fixtureconsts.ServiceAccount3, ClusterId: clusters[0]},
				{Id: fixtureconsts.ServiceAccount4, ClusterId: fixtureconsts.ClusterFake2},
			},
			expectedSADeletions: set.NewFrozenStringSet(fixtureconsts.ServiceAccount2, fixtureconsts.ServiceAccount4),
		},
		{
			name:          "Removing when there are no valid clusters",
			validClusters: []string{},
			serviceAccts: []*storage.ServiceAccount{
				{Id: fixtureconsts.ServiceAccount1, ClusterId: clusters[0]},
				{Id: fixtureconsts.ServiceAccount2, ClusterId: fixtureconsts.ClusterFake1},
				{Id: fixtureconsts.ServiceAccount3, ClusterId: clusters[0]},
				{Id: fixtureconsts.ServiceAccount4, ClusterId: fixtureconsts.ClusterFake2},
			},
			expectedSADeletions: set.NewFrozenStringSet(fixtureconsts.ServiceAccount1, fixtureconsts.ServiceAccount2, fixtureconsts.ServiceAccount3, fixtureconsts.ServiceAccount4),
		},
		{
			name:          "remove K8SRole that belong to deleted clusters",
			validClusters: clusters,
			roles: []*storage.K8SRole{
				{Id: fixtureconsts.Role1, ClusterId: clusters[0]},
				{Id: fixtureconsts.Role2, ClusterId: fixtureconsts.ClusterFake1},
				{Id: fixtureconsts.Role3, ClusterId: clusters[1]},
				{Id: fixtureconsts.Role4, ClusterId: fixtureconsts.ClusterFake2},
			},
			expectedRoleDeletions: set.NewFrozenStringSet(fixtureconsts.Role2, fixtureconsts.Role4),
		},
		{
			name:          "remove K8SRoleBinding that belong to deleted clusters",
			validClusters: clusters,
			bindings: []*storage.K8SRoleBinding{
				{Id: fixtureconsts.RoleBinding1, ClusterId: clusters[0]},
				{Id: fixtureconsts.RoleBinding2, ClusterId: fixtureconsts.ClusterFake1},
				{Id: fixtureconsts.RoleBinding3, ClusterId: clusters[1]},
				{Id: fixtureconsts.RoleBinding4, ClusterId: fixtureconsts.ClusterFake2},
			},
			expectedRBDeletions: set.NewFrozenStringSet(fixtureconsts.RoleBinding2, fixtureconsts.RoleBinding4),
		},
		{
			name:                  "Don't remove anything if all belong to valid cluster",
			validClusters:         clusters,
			serviceAccts:          []*storage.ServiceAccount{{Id: fixtureconsts.ServiceAccount1, ClusterId: clusters[0]}},
			roles:                 []*storage.K8SRole{{Id: fixtureconsts.Role1, ClusterId: clusters[0]}},
			bindings:              []*storage.K8SRoleBinding{{Id: fixtureconsts.RoleBinding1, ClusterId: clusters[0]}},
			expectedSADeletions:   set.NewFrozenStringSet(),
			expectedRoleDeletions: set.NewFrozenStringSet(),
			expectedRBDeletions:   set.NewFrozenStringSet(),
		},
		{
			name:                  "Remove all if they belong to a deleted cluster",
			validClusters:         clusters,
			serviceAccts:          []*storage.ServiceAccount{{Id: fixtureconsts.ServiceAccount1, ClusterId: fixtureconsts.ClusterFake1}},
			roles:                 []*storage.K8SRole{{Id: fixtureconsts.Role1, ClusterId: fixtureconsts.ClusterFake1}},
			bindings:              []*storage.K8SRoleBinding{{Id: fixtureconsts.RoleBinding1, ClusterId: fixtureconsts.ClusterFake1}},
			expectedSADeletions:   set.NewFrozenStringSet(fixtureconsts.ServiceAccount1),
			expectedRoleDeletions: set.NewFrozenStringSet(fixtureconsts.Role1),
			expectedRBDeletions:   set.NewFrozenStringSet(fixtureconsts.RoleBinding1),
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			var serviceAccounts serviceAccountDataStore.DataStore
			var k8sRoles k8sRoleDataStore.DataStore
			var k8sRoleBindings k8sRoleBindingDataStore.DataStore
			var err error
			if env.PostgresDatastoreEnabled.BooleanSetting() {
				serviceAccounts, err = serviceAccountDataStore.GetTestPostgresDataStore(t, s.pool)
				assert.NoError(t, err)
				k8sRoles, err = k8sRoleDataStore.GetTestPostgresDataStore(t, s.pool)
				assert.NoError(t, err)
				k8sRoleBindings, err = k8sRoleBindingDataStore.GetTestPostgresDataStore(t, s.pool)
				assert.NoError(t, err)
			} else {
				db, bleveIndex := setupRocksDBAndBleve(t)
				serviceAccounts, err = serviceAccountDataStore.NewForTestOnly(t, db, bleveIndex)
				assert.NoError(t, err)
				k8sRoles, err = k8sRoleDataStore.NewForTestOnly(t, db, bleveIndex)
				assert.NoError(t, err)
				k8sRoleBindings, err = k8sRoleBindingDataStore.NewForTestOnly(t, db, bleveIndex)
				assert.NoError(t, err)
			}

			for _, sa := range c.serviceAccts {
				assert.NoError(t, serviceAccounts.UpsertServiceAccount(pruningCtx, sa))
			}

			for _, r := range c.roles {
				assert.NoError(t, k8sRoles.UpsertRole(pruningCtx, r))
			}

			for _, b := range c.bindings {
				assert.NoError(t, k8sRoleBindings.UpsertRoleBinding(pruningCtx, b))
			}

			gc := &garbageCollectorImpl{
				serviceAccts:    serviceAccounts,
				k8sRoles:        k8sRoles,
				k8sRoleBindings: k8sRoleBindings,
			}

			q := clusterIDsToNegationQuery(set.NewFrozenStringSet(c.validClusters...))
			gc.removeOrphanedServiceAccounts(q)
			gc.removeOrphanedK8SRoles(q)
			gc.removeOrphanedK8SRoleBindings(q)

			for _, sa := range c.serviceAccts {
				_, ok, err := serviceAccounts.GetServiceAccount(pruningCtx, sa.GetId())
				assert.NoError(t, err)
				assert.Equal(t, !c.expectedSADeletions.Contains(sa.GetId()), ok) // should _not_ be found if it was expected to be deleted
			}

			for _, r := range c.roles {
				_, ok, err := k8sRoles.GetRole(pruningCtx, r.GetId())
				assert.NoError(t, err)
				assert.Equal(t, !c.expectedRoleDeletions.Contains(r.GetId()), ok) // should _not_ be found if it was expected to be deleted
			}

			for _, rb := range c.bindings {
				_, ok, err := k8sRoleBindings.GetRoleBinding(pruningCtx, rb.GetId())
				assert.NoError(t, err)
				assert.Equal(t, !c.expectedRBDeletions.Contains(rb.GetId()), ok) // should _not_ be found if it was expected to be deleted
			}
		})
	}
}

func (s *PruningTestSuite) TestRemoveLogImbues() {
	// Implemented in Postgres only
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}

	cases := []struct {
		name                 string
		logImbues            []*storage.LogImbue
		recentlyRun          bool
		expectedLogDeletions set.FrozenStringSet
	}{
		{
			name:        "remove Log Imbues that are old",
			recentlyRun: false,
			logImbues: []*storage.LogImbue{
				{Id: "log-1", Timestamp: timestampNowMinus(0)},
				{Id: "log-2", Timestamp: timestampNowMinus(24 * time.Hour)},
				{Id: "log-3", Timestamp: timestampNowMinus(24 * 6 * time.Hour)},
				{Id: "log-4", Timestamp: timestampNowMinus(24 * 7 * time.Hour)},
				{Id: "log-5", Timestamp: timestampNowMinus(24 * 8 * time.Hour)},
			},
			expectedLogDeletions: set.NewFrozenStringSet("log-4", "log-5"),
		},
		{
			name:        "recently run, nothing pruned",
			recentlyRun: true,
			logImbues: []*storage.LogImbue{
				{Id: "log-1", Timestamp: timestampNowMinus(0)},
				{Id: "log-2", Timestamp: timestampNowMinus(24 * time.Hour)},
				{Id: "log-3", Timestamp: timestampNowMinus(24 * 6 * time.Hour)},
				{Id: "log-4", Timestamp: timestampNowMinus(24 * 7 * time.Hour)},
				{Id: "log-5", Timestamp: timestampNowMinus(24 * 8 * time.Hour)},
			},
			expectedLogDeletions: set.NewFrozenStringSet(),
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			logImbueStore := logimbueDataStore.GetTestPostgresDataStore(t, s.pool)

			for _, li := range c.logImbues {
				assert.NoError(t, logImbueStore.Upsert(pruningCtx, li))
			}

			gc := &garbageCollectorImpl{
				logimbueStore: logImbueStore,
			}

			if c.recentlyRun {
				lastLogImbuePruneTime = time.Now()
			} else {
				lastLogImbuePruneTime = time.Now().Add(-24 * time.Hour)
			}

			gc.pruneLogImbues()

			logImbues, err := logImbueStore.GetAll(pruningCtx)
			assert.NoError(t, err)
			for _, li := range logImbues {
				assert.False(t, c.expectedLogDeletions.Contains(li.Id))
			}
		})
	}
}

func testDackBoxInstance(t *testing.T, db *rocksdb.RocksDB, index bleve.Index) (*dackbox.DackBox, indexer.WrapperRegistry, queue.WaitableQueue) {
	indexingQ := queue.NewWaitableQueue()
	dacky, err := dackbox.NewRocksDBDackBox(db, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	reg := indexer.NewWrapperRegistry()
	lazy := indexer.NewLazy(indexingQ, reg, index, dacky.AckIndexed)
	lazy.Start()

	return dacky, reg, indexingQ
}

func getAllAlerts() *v1.Query {
	return search.NewQueryBuilder().AddStrings(
		search.ViolationState,
		storage.ViolationState_ACTIVE.String(),
		storage.ViolationState_RESOLVED.String(),
		storage.ViolationState_ATTEMPTED.String(),
	).ProtoQuery()
}

func resourceWithAccess(access storage.Access, resource permissions.ResourceMetadata) permissions.ResourceWithAccess {
	return permissions.ResourceWithAccess{
		Access:   access,
		Resource: resource,
	}
}
