package datastore

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/image/index"
	"bitbucket.org/stack-rox/apollo/central/image/search"
	"bitbucket.org/stack-rox/apollo/central/image/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	gTypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/suite"
)

func TestImageDataStore(t *testing.T) {
	suite.Run(t, new(ImageDataStoreTestSuite))
}

type ImageDataStoreTestSuite struct {
	suite.Suite

	mockIndexer  *index.MockIndexer
	mockSearcher *search.MockSearcher
	mockStore    *store.MockStore

	datastore DataStore
}

func (suite *ImageDataStoreTestSuite) SetupTest() {
	suite.mockIndexer = &index.MockIndexer{}
	suite.mockSearcher = &search.MockSearcher{}
	suite.mockStore = &store.MockStore{}

	suite.datastore = New(suite.mockStore, suite.mockIndexer, suite.mockSearcher)
}

// Scenario: We have a new image with a sha and no scan or metadata. And no previously matched registry shas.
// Outcome: Image should be upserted and indexed unchanged.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithoutMetadata() {
	image := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha1",
		},
	}

	suite.mockStore.On("GetRegistrySha", "sha1").Return("", false, nil)
	suite.mockStore.On("GetImage", "sha1").Return((*v1.Image)(nil), false, nil)

	suite.mockStore.On("UpsertImage", image).Return(nil)
	suite.mockIndexer.On("AddImage", image).Return(nil)

	err := suite.datastore.UpsertDedupeImage(image)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}

// Scenario: We have a new image with metadata, but its sha and the registry sha do not match.
// Outcome: The sha should be changed to the registry sha, and the mapping added to the store.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithMetadata() {
	newImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha1",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
		},
	}
	upsertedImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha2",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
		},
	}

	suite.mockStore.On("GetImage", "sha1").Return((*v1.Image)(nil), false, nil)
	suite.mockStore.On("GetImage", "sha2").Return((*v1.Image)(nil), false, nil)

	suite.mockStore.On("DeleteImage", "sha1").Return(nil)
	suite.mockIndexer.On("DeleteImage", "sha1").Return(nil)

	suite.mockStore.On("UpsertRegistrySha", "sha1", "sha2").Return(nil)
	suite.mockStore.On("UpsertImage", upsertedImage).Return(nil)
	suite.mockIndexer.On("AddImage", upsertedImage).Return(nil)

	err := suite.datastore.UpsertDedupeImage(newImage)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}

// Scenario: We have a new image without, but it has a registry sha in the store.
// Outcome: It's sha should be changed and upserted to the store.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithoutMetadataAndExistingSha() {
	newImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha1",
		},
	}
	upsertedImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha2",
		},
	}

	suite.mockStore.On("GetRegistrySha", "sha1").Return("sha2", true, nil)
	suite.mockStore.On("GetImage", "sha1").Return((*v1.Image)(nil), false, nil)
	suite.mockStore.On("GetImage", "sha2").Return((*v1.Image)(nil), false, nil)

	suite.mockStore.On("DeleteImage", "sha1").Return(nil)
	suite.mockIndexer.On("DeleteImage", "sha1").Return(nil)

	suite.mockStore.On("UpsertRegistrySha", "sha1", "sha2").Return(nil)
	suite.mockStore.On("UpsertImage", upsertedImage).Return(nil)
	suite.mockIndexer.On("AddImage", upsertedImage).Return(nil)

	err := suite.datastore.UpsertDedupeImage(newImage)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}

// Scenario: We have an image with metadata, but its sha and the registry sha do not match.
// Also, We have an existing image for the registry sha with newer metadata and scan data.
// Outcome: The sha should be changed to the registry sha, the mapping should be added to the store,
// and the image should be upserted and indexed with the newest metadata and scan data.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithMetadataAndExistingNewerImage() {
	newImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha1",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
			Created: &gTypes.Timestamp{
				Seconds: 1,
			},
		},
	}
	existingImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha2",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
			Created: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
		Scan: &v1.ImageScan{
			ScanTime: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
	}
	upsertedImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha2",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
			Created: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
		Scan: &v1.ImageScan{
			ScanTime: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
	}

	suite.mockStore.On("GetImage", "sha1").Return((*v1.Image)(nil), false, nil)
	suite.mockStore.On("GetImage", "sha2").Return(existingImage, true, nil)

	suite.mockStore.On("DeleteImage", "sha1").Return(nil)
	suite.mockIndexer.On("DeleteImage", "sha1").Return(nil)

	suite.mockStore.On("UpsertRegistrySha", "sha1", "sha2").Return(nil)
	suite.mockStore.On("UpsertImage", upsertedImage).Return(nil)
	suite.mockIndexer.On("AddImage", upsertedImage).Return(nil)

	err := suite.datastore.UpsertDedupeImage(newImage)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}

// Scenario: We have an image with scan data, and a mapping from its sha to a registry sha. Also,
// We have an existing image for the registry sha with newer metadata and scan data.
// Outcome: The sha should be changed to the registry sha, the mapping should be added to the store,
// and the image should be upserted and indexed with the newest metadata and scan data.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedByShaMapAndNewImageExists() {
	newImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha1",
		},
		Scan: &v1.ImageScan{
			ScanTime: &gTypes.Timestamp{
				Seconds: 1,
			},
		},
	}
	existingOldImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha1",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha0",
			Created: &gTypes.Timestamp{
				Seconds: 1,
			},
		},
		Scan: &v1.ImageScan{
			ScanTime: &gTypes.Timestamp{
				Seconds: 1,
			},
		},
	}
	existingNewImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha2",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
			Created: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
		Scan: &v1.ImageScan{
			ScanTime: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
	}
	upsertedImage := &v1.Image{
		Name: &v1.ImageName{
			Sha: "sha2",
		},
		Metadata: &v1.ImageMetadata{
			RegistrySha: "sha2",
			Created: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
		Scan: &v1.ImageScan{
			ScanTime: &gTypes.Timestamp{
				Seconds: 2,
			},
		},
	}

	suite.mockStore.On("GetRegistrySha", "sha1").Return("sha2", true, nil)
	suite.mockStore.On("GetImage", "sha1").Return(existingOldImage, true, nil)
	suite.mockStore.On("GetImage", "sha2").Return(existingNewImage, true, nil)

	suite.mockStore.On("DeleteImage", "sha1").Return(nil)
	suite.mockIndexer.On("DeleteImage", "sha1").Return(nil)

	suite.mockStore.On("UpsertRegistrySha", "sha1", "sha2").Return(nil)
	suite.mockStore.On("UpsertImage", upsertedImage).Return(nil)
	suite.mockIndexer.On("AddImage", upsertedImage).Return(nil)

	err := suite.datastore.UpsertDedupeImage(newImage)
	suite.NoError(err)

	suite.mockIndexer.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
	suite.mockStore.AssertExpectations(suite.T())
}
