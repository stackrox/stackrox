package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestImages(t *testing.T) {
	suite.Run(t, new(ImagesTestSuite))
}

type ImagesTestSuite struct {
	suite.Suite
	*InMemoryStore
}

func (suite *ImagesTestSuite) SetupSuite() {
	persistent, err := createBoltDB()
	require.Nil(suite.T(), err)
	suite.InMemoryStore = New(persistent)
}

func (suite *ImagesTestSuite) TeardownSuite() {
	suite.Close()
}

func (suite *ImagesTestSuite) basicStorageTest(updateStore, retrievalStore db.Storage) {
	expectedImages := []*v1.Image{
		{
			Sha:      "sha1",
			Registry: "docker.io",
		},
		{
			Sha:      "sha2",
			Registry: "stackrox.io",
		},
	}

	// Test add
	for _, i := range expectedImages {
		suite.NoError(updateStore.AddImage(i))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, i := range expectedImages {
		suite.Error(updateStore.AddImage(i))
	}

	// Verify add is persisted
	images, err := retrievalStore.GetImages(&v1.GetImagesRequest{})
	suite.NoError(err)
	suite.Equal(expectedImages, images)

	// Verify update works
	for _, image := range expectedImages {
		image.Registry = "dtr.io"
		suite.NoError(updateStore.UpdateImage(image))
	}
	images, err = retrievalStore.GetImages(&v1.GetImagesRequest{})
	suite.NoError(err)
	suite.Equal(expectedImages, images)

	// Verify deletion is persisted
	for _, image := range images {
		suite.NoError(updateStore.RemoveImage(image.Sha))
	}
	images, err = retrievalStore.GetImages(&v1.GetImagesRequest{})
	suite.NoError(err)
	suite.Len(images, 0)
}

func (suite *ImagesTestSuite) TestPersistence() {
	suite.basicStorageTest(suite.InMemoryStore, suite.persistent)
}

func (suite *ImagesTestSuite) TestImages() {
	suite.basicStorageTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *ImagesTestSuite) TestGetImagesFilters() {
	expectedImages := []*v1.Image{
		{Sha: "sha1",
			Registry: "docker.io",
		},
		{
			Sha:      "sha2",
			Registry: "stackrox.io",
		},
	}
	for _, image := range expectedImages {
		suite.NoError(suite.AddImage(image))
	}
	// Get all images
	images, err := suite.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal(expectedImages, images)

	for _, image := range expectedImages {
		suite.NoError(suite.RemoveImage(image.Sha))
	}
}
