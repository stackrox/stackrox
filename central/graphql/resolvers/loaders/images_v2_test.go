package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/imagev2/datastore/mocks"
	imagesView "github.com/stackrox/rox/central/views/images"
	imagesViewMocks "github.com/stackrox/rox/central/views/images/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageV2Loader(t *testing.T) {
	suite.Run(t, new(ImageV2LoaderTestSuite))
}

type ImageV2LoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
	mockView      *imagesViewMocks.MockImageView
}

func (suite *ImageV2LoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
	suite.mockView = imagesViewMocks.NewMockImageView(suite.mockCtrl)
}

func (suite *ImageV2LoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ImageV2LoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded images.
	loader := imageV2LoaderImpl{
		loaded: map[string]*storage.ImageV2{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds:        suite.mockDataStore,
		imageView: suite.mockView,
	}

	// Get a preloaded image from id.
	image, err := loader.FromID(suite.ctx, sha1)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[sha1], image)

	// Get a non-preloaded image from id.
	thirdImage := &storage.ImageV2{Id: sha3}
	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.ImageV2{thirdImage}, nil)

	image, err = loader.FromID(suite.ctx, sha3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), thirdImage, image)

	// Above call should now be preloaded.
	image, err = loader.FromID(suite.ctx, sha3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[sha3], image)
}

func (suite *ImageV2LoaderTestSuite) TestFullImageWithID() {
	// Create a loader with some reloaded images.
	loader := imageV2LoaderImpl{
		loaded: map[string]*storage.ImageV2{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds:        suite.mockDataStore,
		imageView: suite.mockView,
	}

	// Get a preloaded image from id.
	image, err := loader.FullImageWithID(suite.ctx, sha1)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[sha1], image)

	// Get a non-preloaded image from id.
	thirdImageNotFull := &storage.ImageV2{
		Id: sha3,
		ScanStats: &storage.ImageV2_ScanStats{
			ComponentCount: 2,
		},
	}
	thirdImageFull := &storage.ImageV2{
		Id: sha3,
	}

	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.ImageV2{thirdImageNotFull}, nil)
	suite.mockDataStore.EXPECT().GetImagesBatch(suite.ctx, []string{sha3}).
		Return([]*storage.ImageV2{thirdImageFull}, nil)

	image, err = loader.FullImageWithID(suite.ctx, sha3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), thirdImageFull, image)

	// Above call should now be preloaded.
	image, err = loader.FullImageWithID(suite.ctx, sha3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[sha3], image)
}

func (suite *ImageV2LoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded images.
	loader := imageV2LoaderImpl{
		loaded: map[string]*storage.ImageV2{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds:        suite.mockDataStore,
		imageView: suite.mockView,
	}

	// Get a preloaded image from id.
	images, err := loader.FromIDs(suite.ctx, []string{sha1, sha2})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.ImageV2{
		loader.loaded[sha1],
		loader.loaded[sha2],
	}, images)

	// Get a non-preloaded image from id.
	thirdImage := &storage.ImageV2{Id: "sha3"}
	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.ImageV2{thirdImage}, nil)

	images, err = loader.FromIDs(suite.ctx, []string{sha1, sha2, sha3})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.ImageV2{
		loader.loaded[sha1],
		loader.loaded[sha2],
		thirdImage,
	}, images)

	// Above call should now be preloaded.
	images, err = loader.FromIDs(suite.ctx, []string{sha1, sha2, sha3})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.ImageV2{
		loader.loaded[sha1],
		loader.loaded[sha2],
		loader.loaded[sha3],
	}, images)
}

func (suite *ImageV2LoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded images.
	loader := imageV2LoaderImpl{
		loaded: map[string]*storage.ImageV2{
			"sha1": {Id: sha1},
			"sha2": {Id: sha2},
		},
		ds:        suite.mockDataStore,
		imageView: suite.mockView,
	}
	query := &v1.Query{}

	// Get a preloaded image from id.
	results := make([]imagesView.ImageCore, 0)
	core1 := imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core1.EXPECT().GetImageID().Return(sha1)
	results = append(results, core1)

	core2 := imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core2.EXPECT().GetImageID().Return(sha2)
	results = append(results, core2)

	suite.mockView.EXPECT().Get(suite.ctx, query).Return(results, nil)

	images, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.ImageV2{
		loader.loaded[sha1],
		loader.loaded[sha2],
	}, images)

	// Get a non-preloaded image from id.
	results = make([]imagesView.ImageCore, 0)
	core1 = imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core1.EXPECT().GetImageID().Return(sha1)
	results = append(results, core1)

	core2 = imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core2.EXPECT().GetImageID().Return(sha2)
	results = append(results, core2)

	core3 := imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core3.EXPECT().GetImageID().Return(sha3)
	results = append(results, core3)

	suite.mockView.EXPECT().Get(suite.ctx, query).Return(results, nil)

	thirdImage := &storage.ImageV2{Id: "sha3"}
	suite.mockDataStore.EXPECT().GetManyImageMetadata(suite.ctx, []string{sha3}).
		Return([]*storage.ImageV2{thirdImage}, nil)

	images, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.ImageV2{
		loader.loaded[sha1],
		loader.loaded[sha2],
		thirdImage,
	}, images)

	// Above call should now be preloaded.
	results = make([]imagesView.ImageCore, 0)
	core1 = imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core1.EXPECT().GetImageID().Return(sha1)
	results = append(results, core1)

	core2 = imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core2.EXPECT().GetImageID().Return(sha2)
	results = append(results, core2)

	core3 = imagesViewMocks.NewMockImageCore(suite.mockCtrl)
	core3.EXPECT().GetImageID().Return(sha3)
	results = append(results, core3)

	suite.mockView.EXPECT().Get(suite.ctx, query).Return(results, nil)

	images, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.ImageV2{
		loader.loaded[sha1],
		loader.loaded[sha2],
		loader.loaded[sha3],
	}, images)
}
