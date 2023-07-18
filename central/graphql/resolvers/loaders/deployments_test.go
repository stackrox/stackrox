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

const (
	dep1 = "dep1"
	dep2 = "dep2"
	dep3 = "dep3"
)

func TestDeploymentLoader(t *testing.T) {
	suite.Run(t, new(DeploymentLoaderTestSuite))
}

type DeploymentLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *DeploymentLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *DeploymentLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *DeploymentLoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded deployments.
	loader := deploymentLoaderImpl{
		loaded: map[string]*storage.Deployment{
			"dep1": {Id: dep1},
			"dep2": {Id: dep2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded deployment from id.
	deployment, err := loader.FromID(suite.ctx, dep1)
	suite.NoError(err)
	suite.Equal(loader.loaded[dep1], deployment)

	// Get a non-preloaded deployment from id.
	thirdDeployment := &storage.Deployment{Id: dep3}
	suite.mockDataStore.EXPECT().GetDeployments(suite.ctx, []string{dep3}).
		Return([]*storage.Deployment{thirdDeployment}, nil)

	deployment, err = loader.FromID(suite.ctx, dep3)
	suite.NoError(err)
	suite.Equal(thirdDeployment, deployment)

	// Above call should now be preloaded.
	deployment, err = loader.FromID(suite.ctx, dep3)
	suite.NoError(err)
	suite.Equal(loader.loaded[dep3], deployment)
}

func (suite *DeploymentLoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded deployments.
	loader := deploymentLoaderImpl{
		loaded: map[string]*storage.Deployment{
			"dep1": {Id: dep1},
			"dep2": {Id: dep2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded deployment from id.
	deployments, err := loader.FromIDs(suite.ctx, []string{dep1, dep2})
	suite.NoError(err)
	suite.Equal([]*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
	}, deployments)

	// Get a non-preloaded deployment from id.
	thirdDeployment := &storage.Deployment{Id: "dep3"}
	suite.mockDataStore.EXPECT().GetDeployments(suite.ctx, []string{dep3}).
		Return([]*storage.Deployment{thirdDeployment}, nil)

	deployments, err = loader.FromIDs(suite.ctx, []string{dep1, dep2, dep3})
	suite.NoError(err)
	suite.Equal([]*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		thirdDeployment,
	}, deployments)

	// Above call should now be preloaded.
	deployments, err = loader.FromIDs(suite.ctx, []string{dep1, dep2, dep3})
	suite.NoError(err)
	suite.Equal([]*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		loader.loaded[dep3],
	}, deployments)
}

func (suite *DeploymentLoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded deployments.
	loader := deploymentLoaderImpl{
		loaded: map[string]*storage.Deployment{
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
	suite.Equal([]*storage.Deployment{
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

	thirdDeployment := &storage.Deployment{Id: "dep3"}
	suite.mockDataStore.EXPECT().GetDeployments(suite.ctx, []string{dep3}).
		Return([]*storage.Deployment{thirdDeployment}, nil)

	deployments, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		thirdDeployment,
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
	suite.Equal([]*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		loader.loaded[dep3],
	}, deployments)
}
