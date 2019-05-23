package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	searchMock "github.com/stackrox/rox/central/image/datastore/internal/search/mocks"
	storeMock "github.com/stackrox/rox/central/image/datastore/internal/store/mocks"
	indexMock "github.com/stackrox/rox/central/image/index/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestImageDataStore(t *testing.T) {
	suite.Run(t, new(ImageDataStoreTestSuite))
}

type ImageDataStoreTestSuite struct {
	suite.Suite

	mockIndexer  *indexMock.MockIndexer
	mockSearcher *searchMock.MockSearcher
	mockStore    *storeMock.MockStore

	datastore DataStore

	mockCtrl *gomock.Controller
}

func (suite *ImageDataStoreTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockIndexer = indexMock.NewMockIndexer(suite.mockCtrl)
	suite.mockSearcher = searchMock.NewMockSearcher(suite.mockCtrl)
	suite.mockStore = storeMock.NewMockStore(suite.mockCtrl)

	suite.datastore = newDatastoreImpl(suite.mockStore, suite.mockIndexer, suite.mockSearcher)
}

func (suite *ImageDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// Scenario: We have a new image with a sha and no scan or metadata. And no previously matched registry shas.
// Outcome: Image should be upserted and indexed unchanged.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithoutMetadata() {
	image := &storage.Image{
		Id: "sha1",
	}

	suite.mockStore.EXPECT().GetImage("sha1").Return((*storage.Image)(nil), false, nil)

	suite.mockStore.EXPECT().UpsertImage(image).Return(nil)
	suite.mockIndexer.EXPECT().AddImage(image).Return(nil)

	err := suite.datastore.UpsertImage(context.TODO(), image)
	suite.NoError(err)
}

// Scenario: We have a new image with metadata, but its sha and the registry sha do not match.
// Outcome: The sha should be changed to the registry sha, and the mapping added to the store.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithMetadata() {
	newImage := &storage.Image{
		Id:       "sha1",
		Metadata: &storage.ImageMetadata{},
	}
	upsertedImage := &storage.Image{
		Id:       "sha1",
		Metadata: &storage.ImageMetadata{},
	}

	suite.mockStore.EXPECT().GetImage("sha1").Return((*storage.Image)(nil), false, nil)
	suite.mockStore.EXPECT().UpsertImage(upsertedImage).Return(nil)
	suite.mockIndexer.EXPECT().AddImage(upsertedImage).Return(nil)

	err := suite.datastore.UpsertImage(context.TODO(), newImage)
	suite.NoError(err)
}
