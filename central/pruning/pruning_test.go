package pruning

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/alert/convert"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	alertDatastoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageDatastoreMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	processWhitelistDatastoreMocks "github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRetentionResolvedDeploy = 7
	testRetentionAllRuntime     = 6
	testRetentionDeletedRuntime = 3
)

var (
	testConfig = &storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention: &storage.PrivateConfig_AlertConfig{
				AlertConfig: &storage.AlertRetentionConfig{
					AllRuntimeRetentionDurationDays:     testRetentionAllRuntime,
					DeletedRuntimeRetentionDurationDays: testRetentionDeletedRuntime,
					ResolvedDeployRetentionDurationDays: testRetentionResolvedDeploy,
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
	var alertDeployment *storage.Alert_Deployment
	if deployment != nil {
		alertDeployment = convert.ToAlertDeployment(deployment)
	} else {
		alertDeployment = &storage.Alert_Deployment{
			Id:       "inactive",
			Inactive: true,
		}
	}
	return &storage.Alert{
		Id: id,

		LifecycleStage: stage,
		State:          state,
		Deployment:     alertDeployment,
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
		containers = append(containers, &storage.Container{
			Image: &storage.ContainerImage{Id: types.NewDigest(id).Digest()},
		})
	}
	return &storage.Deployment{
		Id:         "id",
		Containers: containers,
	}
}

func generateImageDataStructures(ctx context.Context, t *testing.T) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db := testutils.BadgerDBForT(t)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	// Initialize real datastore
	images, err := imageDatastore.NewBadger(db, bleveIndex, true)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockProcessDataStore := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)
	mockProcessDataStore.EXPECT().RemoveProcessIndicatorsOfStaleContainers(gomock.Any(), gomock.Any(), gomock.Any()).Return((error)(nil))

	mockWhitelistDataStore := processWhitelistDatastoreMocks.NewMockDataStore(ctrl)

	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)

	mockAlertDatastore := alertDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any())
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any())
	deployments, err := deploymentDatastore.NewBadger(db, bleveIndex, nil, mockProcessDataStore, mockWhitelistDataStore, nil, mockRiskDatastore, nil, nil)
	require.NoError(t, err)

	return mockAlertDatastore, mockConfigDatastore, images, deployments
}

func generateAlertDataStructures(ctx context.Context, t *testing.T) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db := testutils.BadgerDBForT(t)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	// Initialize real datastore
	alerts := alertDatastore.NewWithDb(db, bleveIndex)

	ctrl := gomock.NewController(t)
	mockProcessDataStore := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)
	mockProcessDataStore.EXPECT().RemoveProcessIndicatorsOfStaleContainers(gomock.Any(), gomock.Any(), gomock.Any()).Return((error)(nil))

	mockWhitelistDataStore := processWhitelistDatastoreMocks.NewMockDataStore(ctrl)

	mockImageDatastore := imageDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(testConfig, nil)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any())
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any())
	deployments, err := deploymentDatastore.NewBadger(db, bleveIndex, nil, mockProcessDataStore, mockWhitelistDataStore, nil, mockRiskDatastore, nil, nil)
	require.NoError(t, err)

	return alerts, mockConfigDatastore, mockImageDatastore, deployments
}

func TestImagePruning(t *testing.T) {
	var cases = []struct {
		name        string
		images      []*storage.Image
		deployment  *storage.Deployment
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
			name: "one old and one new - no deployments",
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
			name: "one old and one new - 1 deployment with old",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", configDatastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id2"),
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
			alerts, config, images, deployments := generateImageDataStructures(ctx, t)

			gc := newGarbageCollector(alerts, images, deployments, config).(*garbageCollectorImpl)

			// Add images and deployments into the datastores
			if c.deployment != nil {
				require.NoError(t, deployments.UpsertDeployment(ctx, c.deployment))
			}
			for _, image := range c.images {
				image.Id = types.NewDigest(image.Id).Digest()
				require.NoError(t, images.UpsertImage(ctx, image))
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
		name        string
		alerts      []*storage.Alert
		expectedIDs []string
		deployments []*storage.Deployment
	}{
		{
			name: "No pruning",
			alerts: []*storage.Alert{
				newAlertInstance("id1", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{"id1", "id2"},
		},
		{
			name: "One old alert, and one new alert",
			alerts: []*storage.Alert{
				newAlertInstance("id1", 1, storage.LifecycleStage_RUNTIME, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", testRetentionAllRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "One old runtime alert, and one old deploy time unresolved alert",
			alerts: []*storage.Alert{
				newAlertInstance("id1", testRetentionAllRuntime+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", testRetentionAllRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old deploy time alert resolved",
			alerts: []*storage.Alert{
				newAlertInstance("id1", testRetentionResolvedDeploy+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{},
		},
		{
			name: "two old-ish runtime alerts, one with no deployment",
			alerts: []*storage.Alert{
				newAlertInstanceWithDeployment("id1", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED, nil),
				newAlertInstanceWithDeployment("id2", testRetentionDeletedRuntime+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED, existsDeployment),
			},
			expectedIDs: []string{"id2"},
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

			gc := newGarbageCollector(alerts, images, deployments, config).(*garbageCollectorImpl)

			// Add alerts into the datastores
			for _, alert := range c.alerts {
				require.NoError(t, alerts.AddAlert(ctx, alert))
			}
			for _, deployment := range c.deployments {
				require.NoError(t, deployments.UpsertDeployment(ctx, deployment))
			}
			all, err := alerts.Search(ctx, search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_RESOLVED.String()).ProtoQuery())
			if err != nil {
				t.Error(err)
			}
			log.Infof("All query returns %d objects: %v", len(all), search.ResultsToIDs(all))

			conf, err := config.GetConfig(ctx)
			require.NoError(t, err, "failed to get config")

			// Garbage collect all of the alerts
			gc.collectAlerts(conf.GetPrivateConfig())

			q := search.NewQueryBuilder().AddStrings(search.ViolationState,
				storage.ViolationState_ACTIVE.String(), storage.ViolationState_RESOLVED.String()).ProtoQuery()
			// Grab the actual remaining alerts and make sure they match the alerts expected to be remaining
			remainingAlerts, err := alerts.SearchListAlerts(ctx, q)
			require.NoError(t, err)

			log.Infof("Remaining alerts: %v", remainingAlerts)
			var ids []string
			for _, i := range remainingAlerts {
				ids = append(ids, i.GetId())
			}

			assert.ElementsMatch(t, c.expectedIDs, ids)
		})
	}
}
