package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	deploymentsViewMocks "github.com/stackrox/rox/central/views/deployments/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
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
	t.Setenv(features.FlattenCVEData.EnvVar(), "false")
	if features.FlattenCVEData.Enabled() {
		t.Skip("FlattenCVEData is enabled")
	}
	suite.Run(t, new(DeploymentLoaderTestSuite))
}

func TestDeploymentLoaderFlattenedCVEData(t *testing.T) {
	t.Setenv(features.FlattenCVEData.EnvVar(), "true")
	if !features.FlattenCVEData.Enabled() {
		t.Skip("FlattenCVEData is disabled")
	}
	suite.Run(t, new(DeploymentLoaderTestSuite))
}

type DeploymentLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
	mockView      *deploymentsViewMocks.MockDeploymentView
}

func (suite *DeploymentLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
	suite.mockView = deploymentsViewMocks.NewMockDeploymentView(suite.mockCtrl)
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
		ds:             suite.mockDataStore,
		deploymentView: suite.mockView,
	}

	// Get a preloaded deployment from id.
	deployment, err := loader.FromID(suite.ctx, dep1)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[dep1], deployment)

	// Get a non-preloaded deployment from id.
	thirdDeployment := &storage.Deployment{Id: dep3}
	suite.mockDataStore.EXPECT().GetDeployments(suite.ctx, []string{dep3}).
		Return([]*storage.Deployment{thirdDeployment}, nil)

	deployment, err = loader.FromID(suite.ctx, dep3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), thirdDeployment, deployment)

	// Above call should now be preloaded.
	deployment, err = loader.FromID(suite.ctx, dep3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[dep3], deployment)
}

func (suite *DeploymentLoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded deployments.
	loader := deploymentLoaderImpl{
		loaded: map[string]*storage.Deployment{
			"dep1": {Id: dep1},
			"dep2": {Id: dep2},
		},
		ds:             suite.mockDataStore,
		deploymentView: suite.mockView,
	}

	// Get a preloaded deployment from id.
	deployments, err := loader.FromIDs(suite.ctx, []string{dep1, dep2})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
	}, deployments)

	// Get a non-preloaded deployment from id.
	thirdDeployment := &storage.Deployment{Id: "dep3"}
	suite.mockDataStore.EXPECT().GetDeployments(suite.ctx, []string{dep3}).
		Return([]*storage.Deployment{thirdDeployment}, nil)

	deployments, err = loader.FromIDs(suite.ctx, []string{dep1, dep2, dep3})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		thirdDeployment,
	}, deployments)

	// Above call should now be preloaded.
	deployments, err = loader.FromIDs(suite.ctx, []string{dep1, dep2, dep3})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.Deployment{
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
		ds:             suite.mockDataStore,
		deploymentView: suite.mockView,
	}
	query := &v1.Query{}

	// Get a preloaded deployment from id.
	if !features.FlattenCVEData.Enabled() {
		results := []search.Result{
			{
				ID: dep1,
			},
			{
				ID: dep2,
			},
		}
		suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)
	} else {
		results := make([]deploymentsView.DeploymentCore, 0)
		core1 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core1.EXPECT().GetDeploymentID().Return(dep1)
		results = append(results, core1)

		core2 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core2.EXPECT().GetDeploymentID().Return(dep2)
		results = append(results, core2)

		suite.mockView.EXPECT().Get(suite.ctx, query).Return(results, nil)
	}

	deployments, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
	}, deployments)

	// Get a non-preloaded deployment from id.
	if !features.FlattenCVEData.Enabled() {
		results := []search.Result{
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
	} else {
		results := make([]deploymentsView.DeploymentCore, 0)
		core1 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core1.EXPECT().GetDeploymentID().Return(dep1)
		results = append(results, core1)

		core2 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core2.EXPECT().GetDeploymentID().Return(dep2)
		results = append(results, core2)

		core3 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core3.EXPECT().GetDeploymentID().Return(dep3)
		results = append(results, core3)

		suite.mockView.EXPECT().Get(suite.ctx, query).Return(results, nil)
	}

	thirdDeployment := &storage.Deployment{Id: "dep3"}
	suite.mockDataStore.EXPECT().GetDeployments(suite.ctx, []string{dep3}).
		Return([]*storage.Deployment{thirdDeployment}, nil)

	deployments, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		thirdDeployment,
	}, deployments)

	// Above call should now be preloaded.
	if !features.FlattenCVEData.Enabled() {
		results := []search.Result{
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
	} else {
		results := make([]deploymentsView.DeploymentCore, 0)
		core1 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core1.EXPECT().GetDeploymentID().Return(dep1)
		results = append(results, core1)

		core2 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core2.EXPECT().GetDeploymentID().Return(dep2)
		results = append(results, core2)

		core3 := deploymentsViewMocks.NewMockDeploymentCore(suite.mockCtrl)
		core3.EXPECT().GetDeploymentID().Return(dep3)
		results = append(results, core3)

		suite.mockView.EXPECT().Get(suite.ctx, query).Return(results, nil)
	}

	deployments, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.Deployment{
		loader.loaded[dep1],
		loader.loaded[dep2],
		loader.loaded[dep3],
	}, deployments)
}
