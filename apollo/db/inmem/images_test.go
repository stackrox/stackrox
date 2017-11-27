package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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
	image1 := &v1.Image{
		Sha:      "sha1",
		Registry: "docker.io",
	}
	err := updateStore.AddImage(image1)
	suite.Nil(err)

	image2 := &v1.Image{
		Sha:      "sha2",
		Registry: "stackrox.io",
	}
	err = updateStore.AddImage(image2)
	suite.Nil(err)

	// Verify add is persisted
	images, err := retrievalStore.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Image{image1, image2}, images)

	// Verify update works
	image1.Registry = "stackrox.io"
	err = updateStore.UpdateImage(image1)
	suite.Nil(err)
	images, err = retrievalStore.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Image{image1, image2}, images)

	// Verify deletion is persisted
	err = updateStore.RemoveImage(image1.Sha)
	suite.Nil(err)
	err = updateStore.RemoveImage(image2.Sha)
	suite.Nil(err)
	images, err = retrievalStore.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Len(images, 0)
}

func (suite *ImagesTestSuite) TestPersistence() {
	suite.basicStorageTest(suite.InMemoryStore, suite.persistent)
}

func (suite *ImagesTestSuite) TestImages() {
	suite.basicStorageTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *ImagesTestSuite) TestGetImagesFilters() {
	image1 := &v1.Image{
		Sha:      "sha1",
		Registry: "docker.io",
	}
	err := suite.AddImage(image1)
	suite.Nil(err)

	image2 := &v1.Image{
		Sha:      "sha2",
		Registry: "stackrox.io",
	}
	err = suite.AddImage(image2)
	suite.Nil(err)

	// Get all images
	images, err := suite.GetImages(&v1.GetImagesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Image{image1, image2}, images)
}
