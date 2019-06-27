package pruning

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	alertDatastoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	"github.com/stackrox/rox/central/config/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageDatastoreMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	processWhitelistDatastoreMocks "github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAlertInstance(id string, daysOld int, stage storage.LifecycleStage, state storage.ViolationState) *storage.Alert {
	return &storage.Alert{
		Id: id,

		LifecycleStage: stage,
		State:          state,
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
			Image: &storage.ContainerImage{Id: id},
		})
	}
	return &storage.Deployment{
		Id:         "id",
		Containers: containers,
	}
}

func generateImageDataStructures(ctx context.Context, t *testing.T) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db := testutils.DBForT(t)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	// Initialize real datastore
	images, err := imageDatastore.New(db, bleveIndex, true)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockProcessDataStore := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)
	mockProcessDataStore.EXPECT().RemoveProcessIndicatorsOfStaleContainers(gomock.Any(), gomock.Any(), gomock.Any()).Return((error)(nil))

	mockWhitelistDataStore := processWhitelistDatastoreMocks.NewMockDataStore(ctrl)

	mockConfigDatastore := configDatastoreMocks.NewMockDataStore(ctrl)
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(&storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention: &storage.PrivateConfig_AlertRetentionDurationDays{
				AlertRetentionDurationDays: datastore.DefaultAlertRetention,
			},
			ImageRetentionDurationDays: datastore.DefaultImageRetention,
		},
	}, nil)

	mockAlertDatastore := alertDatastoreMocks.NewMockDataStore(ctrl)
	deployments, err := deploymentDatastore.New(db, bleveIndex, nil, mockProcessDataStore, mockWhitelistDataStore, nil)
	require.NoError(t, err)

	return mockAlertDatastore, mockConfigDatastore, images, deployments
}

func generateAlertDataStructures(ctx context.Context, t *testing.T) (alertDatastore.DataStore, configDatastore.DataStore, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db := testutils.DBForT(t)

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
	mockConfigDatastore.EXPECT().GetConfig(ctx).Return(&storage.Config{
		PrivateConfig: &storage.PrivateConfig{
			AlertRetention: &storage.PrivateConfig_AlertRetentionDurationDays{
				AlertRetentionDurationDays: datastore.DefaultAlertRetention,
			},
			ImageRetentionDurationDays: datastore.DefaultImageRetention,
		},
	}, nil)

	deployments, err := deploymentDatastore.New(db, bleveIndex, nil, mockProcessDataStore, mockWhitelistDataStore, nil)
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
				newImageInstance("id2", datastore.DefaultImageRetention+1),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 deployment with new",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", datastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id1"),
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 deployment with old",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", datastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id2"),
			expectedIDs: []string{"id1", "id2"},
		},
		{
			name: "two old - 1 deployment with old",
			images: []*storage.Image{
				newImageInstance("id1", datastore.DefaultImageRetention+1),
				newImageInstance("id2", datastore.DefaultImageRetention+1),
			},
			deployment:  newDeployment("id2"),
			expectedIDs: []string{"id2"},
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
			alerts, config, imageDatastore, deployments := generateImageDataStructures(ctx, t)

			gc := newGarbageCollector(alerts, imageDatastore, deployments, config).(*garbageCollectorImpl)

			// Add images and deployments into the datastores
			if c.deployment != nil {
				require.NoError(t, deployments.UpsertDeployment(ctx, c.deployment))
			}
			for _, image := range c.images {
				require.NoError(t, imageDatastore.UpsertImage(ctx, image))
			}

			conf, err := config.GetConfig(ctx)
			require.NoError(t, err, "failed to get config")
			// Garbage collect all of the images
			gc.collectImages(conf.GetPrivateConfig())

			// Grab the  actual remaining images and make sure they match the images expected to be remaining
			remainingImages, err := imageDatastore.ListImages(ctx)
			require.NoError(t, err)

			var ids []string
			for _, i := range remainingImages {
				ids = append(ids, i.GetId())
			}
			assert.ElementsMatch(t, c.expectedIDs, ids)
		})
	}
}

func TestAlertPruning(t *testing.T) {
	var cases = []struct {
		name        string
		alerts      []*storage.Alert
		expectedIDs []string
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
				newAlertInstance("id2", datastore.DefaultAlertRetention+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "One old runtime alert, and one old deploy time unresolved alert",
			alerts: []*storage.Alert{
				newAlertInstance("id1", datastore.DefaultAlertRetention+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_ACTIVE),
				newAlertInstance("id2", datastore.DefaultAlertRetention+1, storage.LifecycleStage_RUNTIME, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old deploy time alert resolved",
			alerts: []*storage.Alert{
				newAlertInstance("id1", datastore.DefaultAlertRetention+1, storage.LifecycleStage_DEPLOY, storage.ViolationState_RESOLVED),
			},
			expectedIDs: []string{},
		},
	}
	scc := sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Alert, resources.Config, resources.Deployment, resources.Image)),
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Alert, resources.Image)),
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			alerts, config, imageDatastore, deployments := generateAlertDataStructures(ctx, t)

			gc := newGarbageCollector(alerts, imageDatastore, deployments, config).(*garbageCollectorImpl)

			// Add alerts into the datastores
			for _, alert := range c.alerts {
				require.NoError(t, alerts.AddAlert(ctx, alert))
			}

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
