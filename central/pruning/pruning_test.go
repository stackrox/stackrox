package pruning

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	processWhitelistDatastoreMocks "github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			Image: &storage.Image{Id: id},
		})
	}
	return &storage.Deployment{
		Id:         "id",
		Containers: containers,
	}
}

func generateDataStructures(t *testing.T) (imageDatastore.DataStore, deploymentDatastore.DataStore) {
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

	deployments, err := deploymentDatastore.New(db, bleveIndex, mockProcessDataStore, mockWhitelistDataStore, nil)
	require.NoError(t, err)

	return images, deployments
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
				newImageInstance("id2", pruneImagesAfterDays+1),
			},
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 deployment with new",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", pruneImagesAfterDays+1),
			},
			deployment:  newDeployment("sha256:id1"),
			expectedIDs: []string{"id1"},
		},
		{
			name: "one old and one new - 1 deployment with old",
			images: []*storage.Image{
				newImageInstance("id1", 1),
				newImageInstance("id2", pruneImagesAfterDays+1),
			},
			deployment:  newDeployment("sha256:id2"),
			expectedIDs: []string{"id1", "id2"},
		},
		{
			name: "two old - 1 deployment with old",
			images: []*storage.Image{
				newImageInstance("id1", pruneImagesAfterDays+1),
				newImageInstance("id2", pruneImagesAfterDays+1),
			},
			deployment:  newDeployment("sha256:id2"),
			expectedIDs: []string{"id2"},
		},
	}

	scc := sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Deployment, resources.Image),
	)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			imageDatastore, deployments := generateDataStructures(t)

			gc := newGarbageCollector(imageDatastore, deployments).(*garbageCollectorImpl)

			// Add images and deployments into the datastores
			if c.deployment != nil {
				require.NoError(t, deployments.UpsertDeployment(ctx, c.deployment))
			}
			for _, image := range c.images {
				require.NoError(t, imageDatastore.UpsertImage(ctx, image))
			}
			// Garbage collect all of the images
			gc.collectImages()

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
