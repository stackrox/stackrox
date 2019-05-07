package pruning

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	deploymentSearcher "github.com/stackrox/rox/central/deployment/search"
	deploymentStore "github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	imageSearcher "github.com/stackrox/rox/central/image/search"
	imageStore "github.com/stackrox/rox/central/image/store"
	processIndicatorDatastoreMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/protoconv"
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

func generateDataStructures(t *testing.T) (imageStore.Store, imageIndexer.Indexer, imageDatastore.DataStore, deploymentDatastore.DataStore) {
	db, err := bolthelper.NewTemp(t.Name() + ".db")
	require.NoError(t, err)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	// Initialize real indexers and datastores
	imageIndexer := imageIndexer.New(bleveIndex)
	imageStore := imageStore.New(db)
	imageSearcher, err := imageSearcher.New(imageStore, imageIndexer)
	require.NoError(t, err)
	images := imageDatastore.New(imageStore, imageIndexer, imageSearcher)

	deploymentIndexer := deploymentIndexer.New(bleveIndex)
	deploymentStore, err := deploymentStore.New(db)
	require.NoError(t, err)

	deploymentSearcher, err := deploymentSearcher.New(deploymentStore, deploymentIndexer)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockProcessDataStore := processIndicatorDatastoreMocks.NewMockDataStore(ctrl)
	mockProcessDataStore.EXPECT().RemoveProcessIndicatorsOfStaleContainers(gomock.Any(), gomock.Any(), gomock.Any()).Return((error)(nil))

	deployments := deploymentDatastore.New(deploymentStore, deploymentIndexer, deploymentSearcher, mockProcessDataStore, nil)
	return imageStore, imageIndexer, images, deployments
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

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Get all of the image constructs because I update the time within the store
			// So to test need to update them separately
			imageStore, imageIndexer, imageDatastore, deployments := generateDataStructures(t)

			gc := newGarbageCollector(imageDatastore, deployments).(*garbageCollectorImpl)

			// Add images and deployments into the datastores
			for _, image := range c.images {
				require.NoError(t, imageIndexer.AddImage(image))
				require.NoError(t, imageStore.UpsertImage(image))
			}
			if c.deployment != nil {
				require.NoError(t, deployments.UpsertDeployment(context.TODO(), c.deployment))
			}

			// Garbage collect all of the images
			gc.collectImages()

			// Grab the  actual remaining images and make sure they match the images expected to be remaining
			remainingImages, err := imageDatastore.ListImages(context.TODO())
			require.NoError(t, err)

			var ids []string
			for _, i := range remainingImages {
				ids = append(ids, i.GetId())
			}
			assert.ElementsMatch(t, c.expectedIDs, ids)
		})
	}
}
