package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/image/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	imageCve1 = "cve1"
	imageCve2 = "cve2"
	imageCve3 = "cve3"
)

func TestImageCVELoader(t *testing.T) {
	suite.Run(t, new(ImageCVELoaderTestSuite))
}

type ImageCVELoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *ImageCVELoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *ImageCVELoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ImageCVELoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded cves.
	loader := imageCveLoaderImpl{
		loaded: map[string]*storage.ImageCVE{
			"cve1": {Id: imageCve1},
			"cve2": {Id: imageCve2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cve, err := loader.FromID(suite.ctx, imageCve1)
	suite.NoError(err)
	suite.Equal(loader.loaded[imageCve1], cve)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.ImageCVE{Id: imageCve3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{imageCve3}).
		Return([]*storage.ImageCVE{thirdCVE}, nil)

	cve, err = loader.FromID(suite.ctx, imageCve3)
	suite.NoError(err)
	suite.Equal(thirdCVE, cve)

	// Above call should now be preloaded.
	cve, err = loader.FromID(suite.ctx, imageCve3)
	suite.NoError(err)
	suite.Equal(loader.loaded[imageCve3], cve)
}

func (suite *ImageCVELoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded cves.
	loader := imageCveLoaderImpl{
		loaded: map[string]*storage.ImageCVE{
			"cve1": {Id: imageCve1},
			"cve2": {Id: imageCve2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cves, err := loader.FromIDs(suite.ctx, []string{imageCve1, imageCve2})
	suite.NoError(err)
	suite.Equal([]*storage.ImageCVE{
		loader.loaded[imageCve1],
		loader.loaded[imageCve2],
	}, cves)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.ImageCVE{Id: "cve3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{imageCve3}).
		Return([]*storage.ImageCVE{thirdCVE}, nil)

	cves, err = loader.FromIDs(suite.ctx, []string{imageCve1, imageCve2, imageCve3})
	suite.NoError(err)
	suite.Equal([]*storage.ImageCVE{
		loader.loaded[imageCve1],
		loader.loaded[imageCve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	cves, err = loader.FromIDs(suite.ctx, []string{imageCve1, imageCve2, imageCve3})
	suite.NoError(err)
	suite.Equal([]*storage.ImageCVE{
		loader.loaded[imageCve1],
		loader.loaded[imageCve2],
		loader.loaded[imageCve3],
	}, cves)
}

func (suite *ImageCVELoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded cves.
	loader := imageCveLoaderImpl{
		loaded: map[string]*storage.ImageCVE{
			"cve1": {Id: imageCve1},
			"cve2": {Id: imageCve2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded cve from id.
	results := []search.Result{
		{
			ID: imageCve1,
		},
		{
			ID: imageCve2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ImageCVE{
		loader.loaded[imageCve1],
		loader.loaded[imageCve2],
	}, cves)

	// Get a non-preloaded cve from id.
	results = []search.Result{
		{
			ID: imageCve1,
		},
		{
			ID: imageCve2,
		},
		{
			ID: imageCve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdCVE := &storage.ImageCVE{Id: "cve3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{imageCve3}).
		Return([]*storage.ImageCVE{thirdCVE}, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ImageCVE{
		loader.loaded[imageCve1],
		loader.loaded[imageCve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: imageCve1,
		},
		{
			ID: imageCve2,
		},
		{
			ID: imageCve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ImageCVE{
		loader.loaded[imageCve1],
		loader.loaded[imageCve2],
		loader.loaded[imageCve3],
	}, cves)
}
