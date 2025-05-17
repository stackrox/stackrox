//go:build sql_integration

package service

import (
	"context"
	"testing"

	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopDataStore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestPLOPService(t *testing.T) {
	suite.Run(t, new(PLOPServiceTestSuite))
}

type PLOPServiceTestSuite struct {
	suite.Suite
	datastore          plopDataStore.DataStore
	store              plopStore.Store
	indicatorDataStore processIndicatorDataStore.DataStore
	service            *serviceImpl

	postgres *pgtest.TestPostgres

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context
	hasAllCtx   context.Context
}

func (suite *PLOPServiceTestSuite) SetupSuite() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))

	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))

	suite.hasAllCtx = sac.WithAllAccess(context.Background())
}

func (suite *PLOPServiceTestSuite) SetupTest() {
	suite.postgres = pgtest.ForT(suite.T())
	suite.store = postgresStore.NewFullStore(suite.postgres.DB)

	indicatorStorage := processIndicatorStorage.New(suite.postgres.DB)
	indicatorSearcher := processIndicatorSearch.New(indicatorStorage)

	suite.indicatorDataStore = processIndicatorDataStore.New(
		indicatorStorage, suite.store, indicatorSearcher, nil)
	suite.datastore = plopDataStore.New(suite.store, suite.indicatorDataStore, suite.postgres)
	suite.service = &serviceImpl{
		dataStore: suite.datastore,
	}
}

var (
	indicator1 = &storage.ProcessIndicator{
		Id:            fixtureconsts.ProcessIndicatorID1,
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		ClusterId:     fixtureconsts.Cluster1,
		ContainerName: "test_container1",
		Namespace:     fixtureconsts.Namespace1,
		Signal: &storage.ProcessSignal{
			Name:         "test_process1",
			Args:         "test_arguments1",
			ExecFilePath: "test_path1",
		},
	}

	indicator2 = &storage.ProcessIndicator{
		Id:            fixtureconsts.ProcessIndicatorID2,
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName3,
		PodUid:        fixtureconsts.PodUID3,
		ClusterId:     fixtureconsts.Cluster1,
		ContainerName: "test_container2",
		Namespace:     fixtureconsts.Namespace1,

		Signal: &storage.ProcessSignal{
			Name:         "test_process2",
			Args:         "test_arguments2",
			ExecFilePath: "test_path2",
		},
	}

	indicator3 = &storage.ProcessIndicator{
		Id:            fixtureconsts.ProcessIndicatorID3,
		DeploymentId:  fixtureconsts.Deployment1,
		PodId:         fixtureconsts.PodName3,
		PodUid:        fixtureconsts.PodUID3,
		ClusterId:     fixtureconsts.Cluster1,
		ContainerName: "test_container2",
		Namespace:     fixtureconsts.Namespace1,

		Signal: &storage.ProcessSignal{
			Name:         "test_process3",
			Args:         "test_arguments3",
			ExecFilePath: "test_path3",
		},
	}

	deployment1 = &storage.Deployment{Id: fixtureconsts.Deployment1, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}
	deployment2 = &storage.Deployment{Id: fixtureconsts.Deployment2, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}
)

func (suite *PLOPServiceTestSuite) TestPLOPCases() {
	cases := map[string]struct {
		plopsInDB         []*storage.ProcessListeningOnPortStorage
		processIndicators []*storage.ProcessIndicator
		deployments       []*storage.Deployment
		// For now we don't know which PLOP will be returned when doing pagination
		// so we just check the number of PLOPs returned. When sorting is added
		// we will also check the values. Add the sorting ticket here before merging.
		expectedPlopCount int
		expectedTotalListeningEndpoints int32
		request           *v1.GetProcessesListeningOnPortsRequest
	}{
		"One plop is retrieved": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2},
			deployments:       []*storage.Deployment{deployment1, deployment2},
			expectedPlopCount: 1,
			expectedTotalListeningEndpoints: 1,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
			},
		},
		"No plops are retrieved since the deployment is wrong": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2},
			deployments:       []*storage.Deployment{deployment1, deployment2},
			expectedPlopCount: 0,
			expectedTotalListeningEndpoints: 0,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment2,
			},
		},
		"Multiple plops are retrieved": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 3,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
			},
		},
		"One plop is retrieved due to pagination": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 1,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
				Pagination: &v1.Pagination{
					Limit:  1,
					Offset: 0,
				},
			},
		},
		"Two plops are retrieved due to pagination": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 2,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
				Pagination: &v1.Pagination{
					Limit:  2,
					Offset: 0,
				},
			},
		},
		"Limit is greater than the number of plops": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 3,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
				Pagination: &v1.Pagination{
					Limit:  4,
					Offset: 0,
				},
			},
		},
		"Limit and offset are one": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 1,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
				Pagination: &v1.Pagination{
					Limit:  1,
					Offset: 1,
				},
			},
		},
		"Two plops returned due to offset": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 2,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
				Pagination: &v1.Pagination{
					Limit:  10,
					Offset: 1,
				},
			},
		},
		"Only one plop returned due to offset": {
			plopsInDB:         []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage7(), fixtures.GetPlopStorage8(), fixtures.GetPlopStorage9()},
			processIndicators: []*storage.ProcessIndicator{indicator1, indicator2, indicator3},
			deployments:       []*storage.Deployment{deployment1},
			expectedPlopCount: 1,
			expectedTotalListeningEndpoints: 3,
			request: &v1.GetProcessesListeningOnPortsRequest{
				DeploymentId: fixtureconsts.Deployment1,
				Pagination: &v1.Pagination{
					Limit:  10,
					Offset: 2,
				},
			},
		},
	}

	for name, c := range cases {
		suite.T().Run(name, func(t *testing.T) {
			suite.SetupTest()

			deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
			suite.Nil(err)

			for _, deployment := range c.deployments {
				suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, deployment))
			}

			suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
				suite.hasWriteCtx, c.processIndicators...))

			err = suite.store.UpsertMany(suite.hasWriteCtx, c.plopsInDB)
			suite.Nil(err)

			response, err := suite.service.GetListeningEndpoints(suite.hasReadCtx, c.request)
			suite.NoError(err)

			suite.Equal(c.expectedPlopCount, len(response.ListeningEndpoints))
			suite.Equal(c.expectedTotalListeningEndpoints, response.TotalListeningEndpoints)
		})
	}

}
