package pruning

import (
	"context"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	alertDatastoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterIndex "github.com/stackrox/rox/central/cluster/index"
	clusterRocksDB "github.com/stackrox/rox/central/cluster/store/cluster/rocksdb"
	clusterHealthRocksDB "github.com/stackrox/rox/central/cluster/store/clusterhealth/rocksdb"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageDatastoreMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentsMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	netEntityMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	networkFlowDatastoreMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	dackboxNodeDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	dackboxNodeGlobalDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	nodeGlobalDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/processor/mocks"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	podMocks "github.com/stackrox/rox/central/pod/datastore/mocks"
	processBaselineDatastoreMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
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
	graphMocks "github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	filterMocks "github.com/stackrox/rox/pkg/process/filter/mocks"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
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
				Id:       "inactive",
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
		Id:         "id",
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
			Id:                  "id",
			LiveInstances:       instances,
			TerminatedInstances: instanceLists,
		}
	}

	return &storage.Pod{
		Id:                  "id",
		TerminatedInstances: instanceLists,
	}
}

func setupRocksDBAndBleve(t *testing.T) (*rocksdb.RocksDB, bleve.Index) {
	db := rocksdbtest.RocksDBForT(t)
	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	return db, bleveIndex
}

func generateImageDataStructures(ctx context.Context, t *testing.T) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore, podDatastore.DataStore, queue.WaitableQueue) {
	db, bleveIndex := setupRocksDBAndBleve(t)

	ctrl := gomock.NewController(t)
	mockComponentDatastore := componentsMocks.NewMockDataStore(ctrl)
	mockComponentDatastore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes()
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	dacky, registry, indexingQ := testDackBoxInstance(t, db, bleveIndex)
	registry.RegisterWrapper(deploymentDackBox.Bucket, deploymentIndex.Wrapper{})
	registry.RegisterWrapper(imageDackBox.Bucket, imageIndex.Wrapper{})

	// Initialize real datastore
	images := imageDatastore.New(dacky, concurrency.NewKeyFence(), bleveIndex, bleveIndex, true, mockRiskDatastore, ranking.NewRanker(), ranking.NewRanker())

	mockProcessDataStore := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)

	mockBaselineDataStore := processBaselineDatastoreMocks.NewMockDataStore(ctrl)

	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)

	mockAlertDatastore := alertDatastoreMocks.NewMockDataStore(ctrl)

	mockFilter := filterMocks.NewMockFilter(ctrl)
	mockFilter.EXPECT().UpdateByPod(gomock.Any()).AnyTimes()

	var pool *pgxpool.Pool
	if features.PostgresDatastore.Enabled() {
		pool = globaldb.GetPostgres()
	}
	deployments := deploymentDatastore.New(dacky, concurrency.NewKeyFence(), pool, nil, bleveIndex, bleveIndex, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, mockFilter, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

	pods, err := podDatastore.NewRocksDB(db, bleveIndex, mockProcessDataStore, mockFilter)
	require.NoError(t, err)

	return mockAlertDatastore, mockConfigDatastore, images, deployments, pods, indexingQ
}

func generateNodeDataStructures(t *testing.T) nodeGlobalDatastore.GlobalDataStore {
	db, bleveIndex := setupRocksDBAndBleve(t)

	ctrl := gomock.NewController(t)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	nodes, err := dackboxNodeGlobalDatastore.New(dackboxNodeDatastore.New(dacky, concurrency.NewKeyFence(), bleveIndex, mockRiskDatastore, ranking.NewRanker(), ranking.NewRanker()))
	require.NoError(t, err)

	return nodes
}

func generateAlertDataStructures(ctx context.Context, t *testing.T) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db, bleveIndex := setupRocksDBAndBleve(t)

	dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	// Initialize real datastore
	alerts := alertDatastore.NewWithDb(db, bleveIndex)

	ctrl := gomock.NewController(t)

	mockBaselineDataStore := processBaselineDatastoreMocks.NewMockDataStore(ctrl)

	mockImageDatastore := imageDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)

	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)

	var pool *pgxpool.Pool
	if features.PostgresDatastore.Enabled() {
		pool = globaldb.GetPostgres()
	}
	deployments := deploymentDatastore.New(dacky, concurrency.NewKeyFence(), pool, nil, bleveIndex, bleveIndex, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

	return alerts, mockConfigDatastore, mockImageDatastore, deployments
}

func generateClusterDataStructures(t *testing.T, config *storage.Config) GarbageCollector {
	db, bleveIndex := setupRocksDBAndBleve(t)

	clusterStorage, err := clusterRocksDB.New(db)
	require.NoError(t, err)

	clusterHealthStorage, err := clusterHealthRocksDB.New(db)
	require.NoError(t, err)

	clusterIndexer := clusterIndex.New(bleveIndex)

	mockCtrl := gomock.NewController(t)
	alertDataStore := alertDatastoreMocks.NewMockDataStore(mockCtrl)
	namespaceDataStore := namespaceMocks.NewMockDataStore(mockCtrl)
	deploymentDataStore := deploymentMocks.NewMockDataStore(mockCtrl)
	nodeDataStore := nodeDatastoreMocks.NewMockGlobalDataStore(mockCtrl)
	podDataStore := podMocks.NewMockDataStore(mockCtrl)
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

	nodeDataStore.EXPECT().GetAllClusterNodeStores(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	netEntityDataStore.EXPECT().RegisterCluster(gomock.Any(), gomock.Any()).AnyTimes()

	clusterDataStore, err := clusterDatastore.New(
		clusterStorage,
		clusterHealthStorage,
		clusterIndexer,
		alertDataStore,
		namespaceDataStore,
		deploymentDataStore,
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
		networkBaselineMgr )

	require.NoError(t, err)
	flowsDataStore.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(networkFlowDatastoreMocks.NewMockFlowDataStore(mockCtrl), nil)
	connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
	namespaceDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil)
	deploymentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	podDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	alertDataStore.EXPECT().SearchRawAlerts(gomock.Any(), gomock.Any()).Return(nil, nil)
	alertDataStore.EXPECT().MarkAlertStale(gomock.Any(), gomock.Any()).Return(nil)
	notifierMock.EXPECT().ProcessAlert(gomock.Any(), gomock.Any()).Return()
	deploymentDataStore.EXPECT().RemoveDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	podDataStore.EXPECT().RemovePod(gomock.Any(), gomock.Any()).Return(nil)
	nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(nil, nil)
	serviceAccountMockDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	roleDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	roleBindingDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	netEntityDataStore.EXPECT().DeleteExternalNetworkEntitiesForCluster(gomock.Any(), gomock.Any()).Return(nil)
	networkBaselineMgr.EXPECT().ProcessPostClusterDelete(gomock.Any()).Return(nil)
	secretDataStore.EXPECT().RemoveSecret(gomock.Any(), gomock.Any()).Return(nil)
	serviceAccountMockDataStore.EXPECT().RemoveServiceAccount(gomock.Any(), gomock.Any()).Return(nil)
	roleDataStore.EXPECT().RemoveRole(gomock.Any(), gomock.Any()).Return(nil)
	roleBindingDataStore.EXPECT().RemoveRoleBinding(gomock.Any(), gomock.Any()).Return(nil)

	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(mockCtrl)
	mockConfigDatastore.EXPECT().GetConfig(gomock.Any()).Return(config, nil)

	gc := newGarbageCollector(
		nil,
		nil,
		nil,
		clusterDataStore,
		nil,
		nil,
		nil,
		nil,
		nil,
		mockConfigDatastore,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil)

	return gc
}

func TestImagePruning(t *testing.T) {
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
				Id: "d1",
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
				newImageInstance("id1", configDatastore.DefaultImageRetention+1),
			},
			pod:         newPod(false, "id1"),
			expectedIDs: []string{},
		},
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			alerts, config, images, deployments, pods, indexQ := generateImageDataStructures(ctx, t)
			nodes := generateNodeDataStructures(t)

			gc := newGarbageCollector(alerts, nodes, images, nil, deployments, pods, nil, nil, nil, config, nil, nil, nil, nil, nil, nil).(*garbageCollectorImpl)

			// Add images, deployments, and pods into the datastores
			if c.deployment != nil {
				require.NoError(t, deployments.UpsertDeployment(ctx, c.deployment))
			}
			if c.pod != nil {
				require.NoError(t, pods.UpsertPod(ctx, c.pod))
			}
			for _, image := range c.images {
				image.Id = types.NewDigest(image.Id).Digest()
				require.NoError(t, images.UpsertImage(ctx, image))
			}

			indexingDone := concurrency.NewSignal()
			indexQ.PushSignal(&indexingDone)
			indexingDone.Wait()

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
		})
	}
}

func TestClusterPruning(t *testing.T) {
	var cases = []struct {
		name        string
		config 	    *storage.Config
		clusters    []*storage.Cluster
		expectedIDs []string
	}{
		{
			name: "1 healthy cluster, 3 unhealthy clusters (1 excluded, 1 unhealthy since few days, 1 unhealthy since many days)",
			config: getTestSystemConfig(90, 72),
			clusters: []*storage.Cluster {
				{
					Id: "HEALTHY cluster",
					HealthStatus: &storage.ClusterHealthStatus{
						SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
					},
				},
				{
					Id: "UNHEALTHY cluster matching a label to ignore the cluster",
					Labels: map[string]string{"k2": "v2"},
					HealthStatus: &storage.ClusterHealthStatus{
						SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
						LastContact:        timeBeforeDays(80),
					},
				},
				{
					Id: "UNHEALTHY cluster with fewer than retentionDays since last contact",
					Labels: map[string]string{"k1": "v2"},
					HealthStatus: &storage.ClusterHealthStatus{
						SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
						LastContact:        timeBeforeDays(10),
					},
				},
				{
					Id: "UNHEALTHY cluster with last contact time before config creation time",
					Labels: map[string]string{"k1": "v2"},
					HealthStatus: &storage.ClusterHealthStatus{
						SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
						LastContact:        timeBeforeDays(80),
					},
				},
			},
			expectedIDs : []string {
				"HEALTHY cluster",
				"UNHEALTHY cluster matching a label to ignore the cluster",
				"UNHEALTHY cluster with fewer than retentionDays since last contact"
			},
		},
	}

	scc := sac.TestScopeCheckerCoreFromAccessResourceMap(t,
		[]permissions.ResourceWithAccess{
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Alert),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Config),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Image),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Risk),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
		})

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)



}

func getTestSystemConfig(createdBeforeDays int, lastUpdatedBeforeHours int) *storage.Config {
	return &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			DecommissionedClusterRetention: &storage.DecommissionedClusterRetentionConfig{
				RetentionDurationDays: 60,
				IgnoreClusterLabels: map[string]string{
					"k1": "v1",
					"k2": "v2",
					"k3": "v3",
				},
				LastUpdated: timeBeforeHours(lastUpdatedBeforeHours),
				CreatedAt:   timeBeforeDays(createdBeforeDays),
			},
		},
	}
}

func TestAlertPruning(t *testing.T) {
	existsDeployment := &storage.Deployment{
		Id:        "deploymentId1",
		Name:      "test deployment",
		Namespace: "ns",
		ClusterId: "clusterid",
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
				newAlertInstance("id1", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
				newAlertInstance("id3", 1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ATTEMPTED),
				newAlertInstance("id4", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{"id1", "id2", "id3", "id4"},
		},
		{
			name: "One old alert, and one new alert",
			alerts: []*storage.Alert{
				newAlertInstance("id1", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", testRetentionAllRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDsRemaining: []string{"id1"},
		},
		{
			name: "One old runtime alert, and one old deploy time unresolved alert",
			alerts: []*storage.Alert{
				newAlertInstance("id1", testRetentionAllRuntime+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", testRetentionAllRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDsRemaining: []string{"id1"},
		},
		{
			name: "one old deploy time alert resolved",
			alerts: []*storage.Alert{
				newAlertInstance("id1", testRetentionResolvedDeploy+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED),
			},
			expectedIDsRemaining: []string{},
		},
		{
			name: "two old-ish runtime alerts, one with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment("id1", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED, nil),
				newAlertInstanceWithDeployment("id2", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED, existsDeployment),
			},
			expectedIDsRemaining: []string{"id2"},
			deployments: []*storage.Deployment{
				existsDeployment,
			},
		},
		{
			name: "expired runtime alert with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment("id1", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE, nil),
			},
			expectedIDsRemaining: []string{},
		},
		{
			name: "One old attempted deploy alert, and one new attempted deploy alert",
			alerts: []*storage.Alert{
				newAlertInstance("id1", 1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ATTEMPTED),
				newAlertInstance("id2", testRetentionAttemptedDeploy+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{"id1"},
		},
		{
			name: "Attempted runtime retention > deleted runtime retention",
			alerts: []*storage.Alert{
				newAlertInstance("id1", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
				newAlertInstance("id2", testRetentionDeletedRuntime-1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
				newAlertInstance("id3", testRetentionAttemptedRuntime-1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{"id1", "id2"},
		},
		{
			name: "Attempted runtime alert with no deployment",
			alerts: []*storage.Alert{
				newAlertInstance("id2", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED),
			},
			expectedIDsRemaining: []string{},
		},
		{
			name: "Attempted runtime alerts, one with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment("id1", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED, nil),
				newAlertInstanceWithDeployment("id2", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ATTEMPTED, existsDeployment),
			},
			expectedIDsRemaining: []string{"id2"},
			deployments: []*storage.Deployment{
				existsDeployment,
			},
		},
	}
	scc := sac.TestScopeCheckerCoreFromAccessResourceMap(t,
		[]permissions.ResourceWithAccess{
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Alert),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Config),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_ACCESS, resources.Image),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Alert),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Deployment),
			resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Image),
		})

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			alerts, config, images, deployments := generateAlertDataStructures(ctx, t)
			nodes := generateNodeDataStructures(t)

			gc := newGarbageCollector(alerts, nodes, images, nil, deployments, nil, nil, nil, nil, config, nil, nil, nil, nil, nil, nil).(*garbageCollectorImpl)

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
		})
	}
}

func timestampNowMinus(t time.Duration) *protoTypes.Timestamp {
	return protoconv.ConvertTimeToTimestamp(time.Now().Add(-t))
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

func TestRemoveOrphanedProcesses(t *testing.T) {
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
				newIndicatorWithDeploymentAndPod("pi1", 1*time.Hour, "dep1", "pod1"),
				newIndicatorWithDeploymentAndPod("pi2", 1*time.Hour, "dep2", "pod2"),
				newIndicatorWithDeploymentAndPod("pi3", 1*time.Hour, "dep3", "pod3"),
			},
			deployments:       set.NewFrozenStringSet(),
			pods:              set.NewFrozenStringSet(),
			expectedDeletions: []string{"pi1", "pi2", "pi3"},
		},
		{
			name: "no deployments nor pods - remove no new orphaned indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod("pi1", 20*time.Minute, "dep1", "pod1"),
				newIndicatorWithDeploymentAndPod("pi2", 20*time.Minute, "dep2", "pod2"),
				newIndicatorWithDeploymentAndPod("pi3", 20*time.Minute, "dep3", "pod3"),
			},
			deployments:       set.NewFrozenStringSet(),
			pods:              set.NewFrozenStringSet(),
			expectedDeletions: []string{},
		},
		{
			name: "all pods separate deployments - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod("pi1", 1*time.Hour, "dep1", "pod1"),
				newIndicatorWithDeploymentAndPod("pi2", 1*time.Hour, "dep2", "pod2"),
				newIndicatorWithDeploymentAndPod("pi3", 1*time.Hour, "dep3", "pod3"),
			},
			deployments:       set.NewFrozenStringSet("dep1", "dep2", "dep3"),
			pods:              set.NewFrozenStringSet("pod1", "pod2", "pod3"),
			expectedDeletions: []string{},
		},
		{
			name: "all pods same deployment - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod("pi1", 1*time.Hour, "dep1", "pod1"),
				newIndicatorWithDeploymentAndPod("pi2", 1*time.Hour, "dep1", "pod2"),
				newIndicatorWithDeploymentAndPod("pi3", 1*time.Hour, "dep1", "pod3"),
			},
			deployments:       set.NewFrozenStringSet("dep1"),
			pods:              set.NewFrozenStringSet("pod1", "pod2", "pod3"),
			expectedDeletions: []string{},
		},
		{
			name: "some pods separate deployments - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod("pi1", 1*time.Hour, "dep1", "pod1"),
				newIndicatorWithDeploymentAndPod("pi2", 20*time.Minute, "dep2", "pod2"),
				newIndicatorWithDeploymentAndPod("pi3", 1*time.Hour, "dep3", "pod3"),
			},
			deployments:       set.NewFrozenStringSet("dep3"),
			pods:              set.NewFrozenStringSet("pod3"),
			expectedDeletions: []string{"pi1"},
		},
		{
			name: "some pods same deployment - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod("pi1", 1*time.Hour, "dep1", "pod1"),
				newIndicatorWithDeploymentAndPod("pi2", 20*time.Minute, "dep1", "pod2"),
				newIndicatorWithDeploymentAndPod("pi3", 1*time.Hour, "dep1", "pod3"),
			},
			deployments:       set.NewFrozenStringSet("dep1"),
			pods:              set.NewFrozenStringSet("pod3"),
			expectedDeletions: []string{"pi1"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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

func TestMarkOrphanedAlerts(t *testing.T) {
	cases := []struct {
		name              string
		initialAlerts     []*storage.ListAlert
		deployments       set.FrozenStringSet
		expectedDeletions []string
	}{
		{
			name: "no deployments - remove all old alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment("alert1", 1*time.Hour, "dep1", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert2", 1*time.Hour, "dep2", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(),
			expectedDeletions: []string{"alert1", "alert2"},
		},
		{
			name: "no deployments - remove no new orphaned alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment("alert1", 20*time.Minute, "dep1", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert2", 20*time.Minute, "dep2", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert3", 20*time.Minute, "dep3", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet(),
			expectedDeletions: []string{},
		},
		{
			name: "all deployments - remove no alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment("alert1", 1*time.Hour, "dep1", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert2", 1*time.Hour, "dep2", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert3", 1*time.Hour, "dep3", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet("dep1", "dep2", "dep3"),
			expectedDeletions: []string{},
		},
		{
			name: "some deployments - remove some alerts",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment("alert1", 1*time.Hour, "dep1", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert2", 20*time.Minute, "dep2", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert3", 1*time.Hour, "dep3", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet("dep3"),
			expectedDeletions: []string{"alert1"},
		},
		{
			name: "some deployments - remove some alerts due to stages",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment("alert1", 1*time.Hour, "dep1", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert2", 1*time.Hour, "dep2", storage.LifecycleStage_BUILD, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert3", 1*time.Hour, "dep3", storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
			},
			deployments:       set.NewFrozenStringSet("dep3"),
			expectedDeletions: []string{"alert1"},
		},
		{
			name: "some deployments - remove some alerts due to state",
			initialAlerts: []*storage.ListAlert{
				newListAlertWithDeployment("alert1", 1*time.Hour, "dep1", storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newListAlertWithDeployment("alert2", 1*time.Hour, "dep2", storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED),
				newListAlertWithDeployment("alert3", 1*time.Hour, "dep3", storage.LifecycleStage_DEPLOY, storage.ViolationState_SNOOZED),
			},
			deployments:       set.NewFrozenStringSet("dep3"),
			expectedDeletions: []string{"alert1"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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
			for _, a := range c.expectedDeletions {
				alerts.EXPECT().MarkAlertStale(pruningCtx, a)
			}
			gci.markOrphanedAlertsAsResolved(c.deployments)
		})
	}
}

func TestRemoveOrphanedNetworkFlows(t *testing.T) {
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
							Id:   "dep1",
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "dep2",
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
							Id:   "dep1",
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "dep2",
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
							Id:   "dep1",
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "dep2",
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet("dep1", "dep2"),
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
							Id:   "dep1",
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "dep2",
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet("dep1"),
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
							Id:   "dep1",
						},
						DstEntity: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "dep2",
						},
					},
				},
			},
			deployments:      set.NewFrozenStringSet("dep2"),
			expectedDeletion: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			clusterFlows := networkFlowDatastoreMocks.NewMockClusterDataStore(ctrl)
			flows := networkFlowDatastoreMocks.NewMockFlowDataStore(ctrl)

			clusterFlows.EXPECT().GetFlowStore(pruningCtx, "cluster").Return(flows, nil)

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
			gci.removeOrphanedNetworkFlows(c.deployments, set.NewFrozenStringSet("cluster"))
		})
	}
}

func TestRemoveOrphanedImageRisks(t *testing.T) {
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
		t.Run(c.name, func(t *testing.T) {
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

func TestRemoveOrphanedNodeRisks(t *testing.T) {
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
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nodes := nodeDatastoreMocks.NewMockGlobalDataStore(ctrl)
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

func TestRemoveOrphanedRBACObjects(t *testing.T) {
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
				{Id: "sa-0", ClusterId: clusters[0]},
				{Id: "sa-1", ClusterId: "invalid-1"},
				{Id: "sa-2", ClusterId: clusters[1]},
				{Id: "sa-3", ClusterId: "invalid-2"},
			},
			expectedSADeletions: set.NewFrozenStringSet("sa-1", "sa-3"),
		},
		{
			name:          "Removing when there is only one valid cluster",
			validClusters: clusters[:1],
			serviceAccts: []*storage.ServiceAccount{
				{Id: "sa-0", ClusterId: clusters[0]},
				{Id: "sa-1", ClusterId: "invalid-1"},
				{Id: "sa-2", ClusterId: clusters[0]},
				{Id: "sa-3", ClusterId: "invalid-2"},
			},
			expectedSADeletions: set.NewFrozenStringSet("sa-1", "sa-3"),
		},
		{
			name:          "Removing when there are no valid clusters",
			validClusters: []string{},
			serviceAccts: []*storage.ServiceAccount{
				{Id: "sa-0", ClusterId: clusters[0]},
				{Id: "sa-1", ClusterId: "invalid-1"},
				{Id: "sa-2", ClusterId: clusters[0]},
				{Id: "sa-3", ClusterId: "invalid-2"},
			},
			expectedSADeletions: set.NewFrozenStringSet("sa-0", "sa-1", "sa-2", "sa-3"),
		},
		{
			name:          "remove K8SRole that belong to deleted clusters",
			validClusters: clusters,
			roles: []*storage.K8SRole{
				{Id: "r-0", ClusterId: clusters[0]},
				{Id: "r-1", ClusterId: "invalid-1"},
				{Id: "r-2", ClusterId: clusters[1]},
				{Id: "r-3", ClusterId: "invalid-2"},
			},
			expectedRoleDeletions: set.NewFrozenStringSet("r-1", "r-3"),
		},
		{
			name:          "remove K8SRoleBinding that belong to deleted clusters",
			validClusters: clusters,
			bindings: []*storage.K8SRoleBinding{
				{Id: "rb-0", ClusterId: clusters[0]},
				{Id: "rb-1", ClusterId: "invalid-1"},
				{Id: "rb-2", ClusterId: clusters[1]},
				{Id: "rb-3", ClusterId: "invalid-2"},
			},
			expectedRBDeletions: set.NewFrozenStringSet("rb-1", "rb-3"),
		},
		{
			name:                  "Don't remove anything if all belong to valid cluster",
			validClusters:         clusters,
			serviceAccts:          []*storage.ServiceAccount{{Id: "sa-0", ClusterId: clusters[0]}},
			roles:                 []*storage.K8SRole{{Id: "r-0", ClusterId: clusters[0]}},
			bindings:              []*storage.K8SRoleBinding{{Id: "rb-0", ClusterId: clusters[0]}},
			expectedSADeletions:   set.NewFrozenStringSet(),
			expectedRoleDeletions: set.NewFrozenStringSet(),
			expectedRBDeletions:   set.NewFrozenStringSet(),
		},
		{
			name:                  "Remove all if they belong to a deleted cluster",
			validClusters:         clusters,
			serviceAccts:          []*storage.ServiceAccount{{Id: "sa-0", ClusterId: "invalid-1"}},
			roles:                 []*storage.K8SRole{{Id: "r-0", ClusterId: "invalid-1"}},
			bindings:              []*storage.K8SRoleBinding{{Id: "rb-0", ClusterId: "invalid-1"}},
			expectedSADeletions:   set.NewFrozenStringSet("sa-0"),
			expectedRoleDeletions: set.NewFrozenStringSet("r-0"),
			expectedRBDeletions:   set.NewFrozenStringSet("rb-0"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, bleveIndex := setupRocksDBAndBleve(t)
			serviceAccounts, err := serviceAccountDataStore.NewForTestOnly(t, db, bleveIndex)
			assert.NoError(t, err)
			k8sRoles, err := k8sRoleDataStore.NewForTestOnly(t, db, bleveIndex)
			assert.NoError(t, err)
			k8sRoleBindings, err := k8sRoleBindingDataStore.NewForTestOnly(t, db, bleveIndex)
			assert.NoError(t, err)

			for _, s := range c.serviceAccts {
				assert.NoError(t, serviceAccounts.UpsertServiceAccount(pruningCtx, s))
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

			for _, s := range c.serviceAccts {
				_, ok, err := serviceAccounts.GetServiceAccount(pruningCtx, s.GetId())
				assert.NoError(t, err)
				assert.Equal(t, !c.expectedSADeletions.Contains(s.GetId()), ok) // should _not_ be found if it was expected to be deleted
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
