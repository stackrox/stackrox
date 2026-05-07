//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	"github.com/stackrox/rox/central/processbaseline/datastore"
	postgresStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	resultsMocks "github.com/stackrox/rox/central/processbaselineresults/datastore/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/central/reprocessor/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension, resources.Deployment)))

	// Create test baselines with different deployment and cluster IDs
	cluster1      = fixtureconsts.Cluster1
	cluster2      = fixtureconsts.Cluster2
	namespace1    = "namespace1"
	namespace2    = "namespace2"
	deployment1ID = fixtureconsts.Deployment1
	deployment2ID = fixtureconsts.Deployment2
	deployment3ID = fixtureconsts.Deployment3

	// Create and add real deployments to the deployment datastore
	deployment1 = &storage.Deployment{
		Id:        deployment1ID,
		Name:      "test-deployment-1",
		Namespace: namespace1,
		ClusterId: cluster1,
		Containers: []*storage.Container{
			{
				Name: "container1",
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						FullName: "nginx:1.19",
					},
				},
			},
		},
	}

	deployment2 = &storage.Deployment{
		Id:        deployment2ID,
		Name:      "test-deployment-2",
		Namespace: namespace2,
		ClusterId: cluster1,
		Containers: []*storage.Container{
			{
				Name: "container2",
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						FullName: "redis:6.0",
					},
				},
			},
		},
	}

	deployment3 = &storage.Deployment{
		Id:        deployment3ID,
		Name:      "test-deployment-1",
		Namespace: namespace2,
		ClusterId: cluster2,
		Containers: []*storage.Container{
			{
				Name: "container1",
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						FullName: "redis:6.0",
					},
				},
			},
			{
				Name: "container3",
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						FullName: "redis:6.0",
					},
				},
			},
			{
				Name: "container4",
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						FullName: "redis:7.0",
					},
				},
			},
		},
	}

	baseline1 = &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			ClusterId:     cluster1,
			Namespace:     namespace1,
			DeploymentId:  deployment1ID,
			ContainerName: "container1",
		},
	}

	baseline2 = &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			ClusterId:     cluster1,
			Namespace:     namespace2,
			DeploymentId:  deployment2ID,
			ContainerName: "container2",
		},
	}

	baseline3 = &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			ClusterId:     cluster2,
			Namespace:     namespace2,
			DeploymentId:  deployment3ID,
			ContainerName: "container1",
		},
	}

	baseline4 = &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			ClusterId:     cluster2,
			Namespace:     namespace2,
			DeploymentId:  deployment3ID,
			ContainerName: "container3",
		},
	}

	baseline5 = &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			ClusterId:     cluster2,
			Namespace:     namespace2,
			DeploymentId:  deployment3ID,
			ContainerName: "container4",
		},
	}

	noCriteriaError = errors.Wrap(errox.InvalidArgs, "At least one parameter must not be empty or a wild card, not counting container name")
)

func TestProcessBaselineServiceReal(t *testing.T) {
	suite.Run(t, new(ProcessBaselineServiceRealTestSuite))
}

type ProcessBaselineServiceRealTestSuite struct {
	suite.Suite
	baselineDatastore datastore.DataStore
	deploymentDS      deploymentDS.DataStore
	service           Service

	pool postgres.DB

	reprocessor        *mocks.MockLoop
	resultDatastore    *resultsMocks.MockDataStore
	indicatorMockStore *indicatorMocks.MockDataStore
	mockCtrl           *gomock.Controller
	lifecycleManager   *lifecycleMocks.MockManager
}

func (suite *ProcessBaselineServiceRealTestSuite) SetupTest() {
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB

	// Set up baseline datastore
	baselineStore := postgresStore.New(suite.pool)
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.resultDatastore = resultsMocks.NewMockDataStore(suite.mockCtrl)
	suite.resultDatastore.EXPECT().DeleteBaselineResults(gomock.Any(), gomock.Any()).AnyTimes()
	suite.indicatorMockStore = indicatorMocks.NewMockDataStore(suite.mockCtrl)
	suite.baselineDatastore = datastore.New(baselineStore, suite.resultDatastore, suite.indicatorMockStore)

	// Set up real deployment datastore
	var err error
	suite.deploymentDS, err = deploymentDS.GetTestPostgresDataStore(suite.T(), suite.pool)
	suite.Require().NoError(err)

	suite.reprocessor = mocks.NewMockLoop(suite.mockCtrl)
	suite.lifecycleManager = lifecycleMocks.NewMockManager(suite.mockCtrl)
	suite.service = New(suite.baselineDatastore, suite.reprocessor, suite.deploymentDS, suite.lifecycleManager)
}

func (suite *ProcessBaselineServiceRealTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
	suite.pool.Close()
}

func (suite *ProcessBaselineServiceRealTestSuite) TestGetProcessBaselineBulk() {
	// Add deployments to the datastore
	err := suite.deploymentDS.UpsertDeployment(writeCtx, deployment1)
	suite.Require().NoError(err)
	err = suite.deploymentDS.UpsertDeployment(writeCtx, deployment2)
	suite.Require().NoError(err)
	err = suite.deploymentDS.UpsertDeployment(writeCtx, deployment3)
	suite.Require().NoError(err)
	defer func() {
		_ = suite.deploymentDS.RemoveDeployment(writeCtx, cluster1, deployment1ID)
		_ = suite.deploymentDS.RemoveDeployment(writeCtx, cluster1, deployment2ID)
		_ = suite.deploymentDS.RemoveDeployment(writeCtx, cluster2, deployment3ID)
	}()

	baselines := []*storage.ProcessBaseline{baseline1, baseline2, baseline3, baseline4, baseline5}
	fillDB(suite.T(), suite.baselineDatastore, baselines)
	defer emptyDB(suite.T(), suite.baselineDatastore, baselines)

	testCases := []struct {
		name              string
		query             *v1.ProcessBaselineQuery
		expectedBaselines []*storage.ProcessBaseline
		err               error
	}{
		{
			name: "Filter by cluster ID",
			query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1, baseline2},
			err:               nil,
		},
		{
			name: "Filter by namespace",
			query: &v1.ProcessBaselineQuery{
				Namespaces: []string{namespace2},
				ClusterIds: []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline2, baseline3, baseline4, baseline5},
			err:               nil,
		},
		{
			name: "Filter by deployment ID",
			query: &v1.ProcessBaselineQuery{
				DeploymentIds: []string{deployment1ID},
				ClusterIds:    []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1},
			err:               nil,
		},
		{
			name: "Filter by container name container1",
			query: &v1.ProcessBaselineQuery{
				ContainerNames: []string{"container1"},
				ClusterIds:     []string{cluster1, cluster2},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1, baseline3},
			err:               nil,
		},
		{
			name: "Filter by container name container2",
			query: &v1.ProcessBaselineQuery{
				ContainerNames: []string{"container2"},
				ClusterIds:     []string{cluster1, cluster2},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline2},
			err:               nil,
		},
		{
			name: "Filter by container name container3",
			query: &v1.ProcessBaselineQuery{
				ContainerNames: []string{"container3"},
				ClusterIds:     []string{cluster1, cluster2},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline4},
			err:               nil,
		},
		{
			name: "Filter by container name container4",
			query: &v1.ProcessBaselineQuery{
				ContainerNames: []string{"container4"},
				ClusterIds:     []string{cluster1, cluster2},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline5},
			err:               nil,
		},
		{
			name: "Filter by deployment name",
			query: &v1.ProcessBaselineQuery{
				DeploymentNames: []string{"test-deployment-1"},
				ClusterIds:      []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1, baseline3, baseline4, baseline5},
			err:               nil,
		},
		{
			name: "Filter by deployment name test-deployment-2",
			query: &v1.ProcessBaselineQuery{
				DeploymentNames: []string{"test-deployment-2"},
				ClusterIds:      []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline2},
			err:               nil,
		},
		{
			name: "Filter by image nginx:1.19",
			query: &v1.ProcessBaselineQuery{
				Images:     []string{"nginx:1.19"},
				ClusterIds: []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1},
			err:               nil,
		},
		{
			name: "Filter by image redis:6.0",
			query: &v1.ProcessBaselineQuery{
				Images:     []string{"redis:6.0"},
				ClusterIds: []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline2, baseline3, baseline4},
			err:               nil,
		},
		{
			name: "Filter by image redis:6.0 and container1",
			query: &v1.ProcessBaselineQuery{
				Images:         []string{"redis:6.0"},
				ContainerNames: []string{"container1"},
				ClusterIds:     []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline3},
			err:               nil,
		},
		{
			name: "Filter by cluster and namespace",
			query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1},
				Namespaces: []string{namespace1},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1},
			err:               nil,
		},
		{
			name: "Filter by deployment name and image",
			query: &v1.ProcessBaselineQuery{
				DeploymentNames: []string{"test-deployment-1"},
				Images:          []string{"nginx:1.19"},
				ClusterIds:      []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1},
			err:               nil,
		},
		{
			name: "Filter by image and container name",
			query: &v1.ProcessBaselineQuery{
				Images:         []string{"nginx:1.19"},
				ContainerNames: []string{"container1"},
				ClusterIds:     []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{baseline1},
			err:               nil,
		},
		{
			name: "No filters returns all baselines",
			query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{"*"},
			},
			expectedBaselines: nil,
			err:               noCriteriaError,
		},
		{
			name: "Filter by non-existent deployment name",
			query: &v1.ProcessBaselineQuery{
				DeploymentNames: []string{"non-existent"},
				ClusterIds:      []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{},
			err:               nil,
		},
		{
			name: "Filter by non-existent image",
			query: &v1.ProcessBaselineQuery{
				Images:     []string{"non-existent:1.0"},
				ClusterIds: []string{"*"},
			},
			expectedBaselines: []*storage.ProcessBaseline{},
			err:               nil,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			request := &v1.GetProcessBaselinesBulkRequest{
				Query: tc.query,
			}

			resp, err := suite.service.GetProcessBaselineBulk(writeCtx, request)
			if tc.err == nil {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}

			protoassert.ElementsMatch(t, tc.expectedBaselines, resp.GetBaselines())
		})
	}
}

func (suite *ProcessBaselineServiceRealTestSuite) TestGetProcessBaselineBulkPagination() {
	// Add deployments to the datastore
	err := suite.deploymentDS.UpsertDeployment(writeCtx, deployment1)
	suite.Require().NoError(err)
	err = suite.deploymentDS.UpsertDeployment(writeCtx, deployment2)
	suite.Require().NoError(err)
	err = suite.deploymentDS.UpsertDeployment(writeCtx, deployment3)
	suite.Require().NoError(err)
	defer func() {
		_ = suite.deploymentDS.RemoveDeployment(writeCtx, cluster1, deployment1ID)
		_ = suite.deploymentDS.RemoveDeployment(writeCtx, cluster1, deployment2ID)
		_ = suite.deploymentDS.RemoveDeployment(writeCtx, cluster2, deployment3ID)
	}()

	baselines := []*storage.ProcessBaseline{baseline1, baseline2, baseline3, baseline4, baseline5}
	fillDB(suite.T(), suite.baselineDatastore, baselines)
	defer emptyDB(suite.T(), suite.baselineDatastore, baselines)

	suite.T().Run("Pagination - offset 0, limit 2", func(t *testing.T) {
		request := &v1.GetProcessBaselinesBulkRequest{
			Query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1, cluster2},
			},
			Pagination: &v1.Pagination{
				Offset: 0,
				Limit:  2,
			},
		}

		resp, err := suite.service.GetProcessBaselineBulk(writeCtx, request)
		suite.NoError(err)
		suite.NotNil(resp)
		suite.Len(resp.GetBaselines(), 2)
		suite.Equal(int32(5), resp.GetTotalCount())
	})

	suite.T().Run("Pagination - offset 1, limit 2", func(t *testing.T) {
		request := &v1.GetProcessBaselinesBulkRequest{
			Query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1, cluster2},
			},
			Pagination: &v1.Pagination{
				Offset: 1,
				Limit:  2,
			},
		}

		resp, err := suite.service.GetProcessBaselineBulk(writeCtx, request)
		suite.NoError(err)
		suite.NotNil(resp)
		suite.Len(resp.GetBaselines(), 2)
		suite.Equal(int32(5), resp.GetTotalCount())
	})

	suite.T().Run("Pagination - offset 2, limit 2", func(t *testing.T) {
		request := &v1.GetProcessBaselinesBulkRequest{
			Query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1, cluster2},
			},
			Pagination: &v1.Pagination{
				Offset: 4,
				Limit:  2,
			},
		}

		resp, err := suite.service.GetProcessBaselineBulk(writeCtx, request)
		suite.NoError(err)
		suite.NotNil(resp)
		suite.Len(resp.GetBaselines(), 1)
		suite.Equal(int32(5), resp.GetTotalCount())
	})

	suite.T().Run("Pagination - offset beyond available results", func(t *testing.T) {
		request := &v1.GetProcessBaselinesBulkRequest{
			Query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1, cluster2},
			},
			Pagination: &v1.Pagination{
				Offset: 10,
				Limit:  2,
			},
		}

		resp, err := suite.service.GetProcessBaselineBulk(writeCtx, request)
		suite.NoError(err)
		suite.NotNil(resp)
		suite.Len(resp.GetBaselines(), 0)
		suite.Equal(int32(5), resp.GetTotalCount())
	})

	suite.T().Run("Pagination - limit only", func(t *testing.T) {
		request := &v1.GetProcessBaselinesBulkRequest{
			Query: &v1.ProcessBaselineQuery{
				ClusterIds: []string{cluster1, cluster2},
			},
			Pagination: &v1.Pagination{
				Offset: 0,
				Limit:  1,
			},
		}

		resp, err := suite.service.GetProcessBaselineBulk(writeCtx, request)
		suite.NoError(err)
		suite.NotNil(resp)
		suite.Len(resp.GetBaselines(), 1)
		suite.Equal(int32(5), resp.GetTotalCount())
	})
}
