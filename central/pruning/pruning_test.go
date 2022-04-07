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
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageDatastoreMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentsMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	networkFlowDatastoreMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	dackboxNodeDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	dackboxNodeGlobalDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	nodeGlobalDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	processBaselineDatastoreMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8sRoleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	riskDatastore "github.com/stackrox/rox/central/risk/datastore"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
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

	deployments := deploymentDatastore.New(dacky, concurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, mockFilter, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

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
	commentsDB := testutils.DBForT(t)

	dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	// Initialize real datastore
	alerts := alertDatastore.NewWithDb(db, commentsDB, bleveIndex)

	ctrl := gomock.NewController(t)

	mockBaselineDataStore := processBaselineDatastoreMocks.NewMockDataStore(ctrl)

	mockImageDatastore := imageDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)

	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)

	deployments := deploymentDatastore.New(dacky, concurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, mockBaselineDataStore, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

	return alerts, mockConfigDatastore, mockImageDatastore, deployments
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

	scc := sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Alert, resources.Config, resources.Deployment, resources.Image, resources.Risk)),
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Alert, resources.Image, resources.Deployment, resources.Risk)),
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

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
	scc := sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Alert, resources.Config, resources.Deployment, resources.Image)),
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Alert, resources.Image, resources.Deployment)),
	}

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
