package datastore

import (
	"testing"

	indexMock "github.com/stackrox/rox/central/image/index/mocks"
	searchMock "github.com/stackrox/rox/central/image/search/mocks"
	storeMock "github.com/stackrox/rox/central/image/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestImageDataStore(t *testing.T) {
	suite.Run(t, new(ImageDataStoreTestSuite))
}

type ImageDataStoreTestSuite struct {
	suite.Suite

	mockIndexer  *indexMock.Indexer
	mockSearcher *searchMock.Searcher
	mockStore    *storeMock.Store

	datastore DataStore
}

func (suite *ImageDataStoreTestSuite) SetupTest() {
	suite.mockIndexer = &indexMock.Indexer{}
	suite.mockSearcher = &searchMock.Searcher{}
	suite.mockStore = &storeMock.Store{}

	suite.datastore = New(suite.mockStore, suite.mockIndexer, suite.mockSearcher)
}

// Scenario: We have a new image with a sha and no scan or metadata. And no previously matched registry shas.
// Outcome: Image should be upserted and indexed unchanged.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithoutMetadata() {
	image := &v1.Image{
		Id: "sha1",
	}

	suite.mockStore.On("GetImage", "sha1").Return((*v1.Image)(nil), false, nil)

	suite.mockStore.On("UpsertImage", image).Return(nil)
	suite.mockIndexer.On("AddImage", image).Return(nil)

	err := suite.datastore.UpsertImage(image)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}

// Scenario: We have a new image with metadata, but its sha and the registry sha do not match.
// Outcome: The sha should be changed to the registry sha, and the mapping added to the store.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithMetadata() {
	newImage := &v1.Image{
		Id:       "sha1",
		Metadata: &v1.ImageMetadata{},
	}
	upsertedImage := &v1.Image{
		Id:       "sha1",
		Metadata: &v1.ImageMetadata{},
	}

	suite.mockStore.On("GetImage", "sha1").Return((*v1.Image)(nil), false, nil)
	suite.mockStore.On("UpsertImage", upsertedImage).Return(nil)
	suite.mockIndexer.On("AddImage", upsertedImage).Return(nil)

	err := suite.datastore.UpsertImage(newImage)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}
