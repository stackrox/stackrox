package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/image/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	sha1 = "sha1"
	sha2 = "sha2"
	sha3 = "sha3"
)

func TestImageLoader(t *testing.T) {
	suite.Run(t, new(ImageLoaderTestSuite))
}

type ImageLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *ImageLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *ImageLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ImageLoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded images.
	loader := imageLoaderImpl{
		loaded: map[string]*storage.Image{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded image from id.
	image, err := loader.FromID(suite.ctx, sha1)
	suite.NoError(err)
	suite.Equal(loader.loaded[sha1], image)

	// Get a non-preloaded image from id.
	thirdImage := &storage.Image{Id: sha3}
	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.Image{thirdImage}, nil)

	image, err = loader.FromID(suite.ctx, sha3)
	suite.NoError(err)
	suite.Equal(thirdImage, image)

	// Above call should now be preloaded.
	image, err = loader.FromID(suite.ctx, sha3)
	suite.NoError(err)
	suite.Equal(loader.loaded[sha3], image)
}

func (suite *ImageLoaderTestSuite) TestFullImageWithID() {
	// Create a loader with some reloaded images.
	loader := imageLoaderImpl{
		loaded: map[string]*storage.Image{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded image from id.
	image, err := loader.FullImageWithID(suite.ctx, sha1)
	suite.NoError(err)
	suite.Equal(loader.loaded[sha1], image)

	// Get a non-preloaded image from id.
	thirdImageNotFull := &storage.Image{
		Id:            sha3,
		SetComponents: &storage.Image_Components{Components: 2},
	}
	thirdImageFull := &storage.Image{
		Id: sha3,
	}

	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.Image{thirdImageNotFull}, nil)
	suite.mockDataStore.EXPECT().GetImagesBatch(suite.ctx, []string{sha3}).
		Return([]*storage.Image{thirdImageFull}, nil)

	image, err = loader.FullImageWithID(suite.ctx, sha3)
	suite.NoError(err)
	suite.Equal(thirdImageFull, image)

	// Above call should now be preloaded.
	image, err = loader.FullImageWithID(suite.ctx, sha3)
	suite.NoError(err)
	suite.Equal(loader.loaded[sha3], image)
}

func (suite *ImageLoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded images.
	loader := imageLoaderImpl{
		loaded: map[string]*storage.Image{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded image from id.
	images, err := loader.FromIDs(suite.ctx, []string{sha1, sha2})
	suite.NoError(err)
	suite.Equal([]*storage.Image{
		loader.loaded[sha1],
		loader.loaded[sha2],
	}, images)

	// Get a non-preloaded image from id.
	thirdImage := &storage.Image{Id: "sha3"}
	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.Image{thirdImage}, nil)

	images, err = loader.FromIDs(suite.ctx, []string{sha1, sha2, sha3})
	suite.NoError(err)
	suite.Equal([]*storage.Image{
		loader.loaded[sha1],
		loader.loaded[sha2],
		thirdImage,
	}, images)

	// Above call should now be preloaded.
	images, err = loader.FromIDs(suite.ctx, []string{sha1, sha2, sha3})
	suite.NoError(err)
	suite.Equal([]*storage.Image{
		loader.loaded[sha1],
		loader.loaded[sha2],
		loader.loaded[sha3],
	}, images)
}

func (suite *ImageLoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded images.
	loader := imageLoaderImpl{
		loaded: map[string]*storage.Image{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded image from id.
	results := []search.Result{
		{
			ID: sha1,
		},
		{
			ID: sha2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	images, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Image{
		loader.loaded[sha1],
		loader.loaded[sha2],
	}, images)

	// Get a non-preloaded image from id.
	results = []search.Result{
		{
			ID: sha1,
		},
		{
			ID: sha2,
		},
		{
			ID: sha3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdImage := &storage.Image{Id: "sha3"}
	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.Image{thirdImage}, nil)

	images, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Image{
		loader.loaded[sha1],
		loader.loaded[sha2],
		thirdImage,
	}, images)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: sha1,
		},
		{
			ID: sha2,
		},
		{
			ID: sha3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	images, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Image{
		loader.loaded[sha1],
		loader.loaded[sha2],
		loader.loaded[sha3],
	}, images)
}
