package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestListDeploymentLoader(t *testing.T) {
	suite.Run(t, new(ListDeploymentLoaderTestSuite))
}

type ListDeploymentLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *ListDeploymentLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *ListDeploymentLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ListDeploymentLoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded deployments.
	loader := listDeploymentLoaderImpl{
		loaded: map[string]*storage.ListDeployment{
			"dep1": {Id: dep1},
			"dep2": {Id: dep2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded deployment from id.
	deployments, err := loader.FromIDs(suite.ctx, []string{dep1, dep2})
	suite.NoError(err)
	suite.Equal([]*storage.ListDeployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
	}, deployments)

	// Get a non-preloaded deployment from id.
	thirdListDeployment := &storage.ListDeployment{Id: "dep3"}
	suite.mockDataStore.EXPECT().SearchListDeployments(suite.ctx, gomock.Any()).
		Return([]*storage.ListDeployment{thirdListDeployment}, nil)

	deployments, err = loader.FromIDs(suite.ctx, []string{dep1, dep2, dep3})
	suite.NoError(err)
	suite.Equal([]*storage.ListDeployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		thirdListDeployment,
	}, deployments)

	// Above call should now be preloaded.
	deployments, err = loader.FromIDs(suite.ctx, []string{dep1, dep2, dep3})
	suite.NoError(err)
	suite.Equal([]*storage.ListDeployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		loader.loaded[dep3],
	}, deployments)
}

func (suite *ListDeploymentLoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded deployments.
	loader := listDeploymentLoaderImpl{
		loaded: map[string]*storage.ListDeployment{
			"dep1": {Id: dep1},
			"dep2": {Id: dep2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded deployment from id.
	results := []search.Result{
		{
			ID: dep1,
		},
		{
			ID: dep2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	deployments, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ListDeployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
	}, deployments)

	// Get a non-preloaded deployment from id.
	results = []search.Result{
		{
			ID: dep1,
		},
		{
			ID: dep2,
		},
		{
			ID: dep3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdListDeployment := &storage.ListDeployment{Id: "dep3"}
	suite.mockDataStore.EXPECT().SearchListDeployments(suite.ctx, gomock.Any()).
		Return([]*storage.ListDeployment{thirdListDeployment}, nil)

	deployments, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ListDeployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		thirdListDeployment,
	}, deployments)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: dep1,
		},
		{
			ID: dep2,
		},
		{
			ID: dep3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	deployments, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ListDeployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		loader.loaded[dep3],
	}, deployments)
}
