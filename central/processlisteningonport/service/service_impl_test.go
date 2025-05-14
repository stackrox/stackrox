//go:build sql_integration

package service

// Uncomment or remove imports before merging
import (
	"context"
	//"fmt"
	//"math/rand"
	//"sync"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	plopDataStore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	//"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	//"github.com/stackrox/rox/pkg/search"
	//"github.com/stackrox/rox/pkg/set"
	//"github.com/stackrox/rox/pkg/uuid"
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

func getIndicators() []*storage.ProcessIndicator {
	indicators := []*storage.ProcessIndicator{
		{
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
		},
		{
			Id:            fixtureconsts.ProcessIndicatorID2,
			DeploymentId:  fixtureconsts.Deployment2,
			PodId:         fixtureconsts.PodName2,
			PodUid:        fixtureconsts.PodUID2,
			ClusterId:     fixtureconsts.Cluster1,
			ContainerName: "test_container2",
			Namespace:     fixtureconsts.Namespace1,

			Signal: &storage.ProcessSignal{
				Name:         "test_process2",
				Args:         "test_arguments2",
				ExecFilePath: "test_path2",
			},
		},
	}
	// Uncomment or remove before merging
	//for _, indicator := range indicators {
	//	id.SetIndicatorID(indicator)
	//}

	return indicators
}

var (
	openPlopObject = storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodUid:       fixtureconsts.PodUID1,
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
	}

	closedPlopObject = storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodUid:       fixtureconsts.PodUID1,
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
	}
)

func (suite *PLOPServiceTestSuite) addDeployments() {
	deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
	suite.Nil(err)
	suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: fixtureconsts.Deployment1, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))
	suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: fixtureconsts.Deployment2, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))
}

func (suite *PLOPServiceTestSuite) TestPLOPCases() {
	cases := map[string]struct{
		plopsInDB		[]*storage.ProcessListeningOnPortStorage
		processIndicators	[]*storage.ProcessIndicator
		deployments		[]*storage.Deployment
		// For now we don't know which PLOP will be returned when doing pagination
		// so we just check the number of PLOPs returned. When sorting is added
		// we will also check the values. Add the sorting ticket here before merging.
		expectedPlopCount      int
		request 		*v1.GetProcessesListeningOnPortsRequest
	} {
		"One plop is retrieved": {
			plopsInDB:	[]*storage.ProcessListeningOnPortStorage{
							fixtures.GetPlopStorage7(),
						},
			processIndicators: getIndicators(),
			deployments: 	[]*storage.Deployment{
						&storage.Deployment{Id: fixtureconsts.Deployment1, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1},
						&storage.Deployment{Id: fixtureconsts.Deployment2, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1},
					},
			expectedPlopCount: 1,
			request:	&v1.GetProcessesListeningOnPortsRequest{
				DeploymentId:	fixtureconsts.Deployment1,
			},
		},
	}
		
	for name, c := range cases {
		suite.T().Run(name, func(t *testing.T) {
			suite.SetupTest()

			deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
			suite.Nil(err)

			for _, deployment := range c.deployments{
				suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, deployment))
			}

			suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
				suite.hasWriteCtx, c.processIndicators...))

			err = suite.store.UpsertMany(suite.hasWriteCtx, c.plopsInDB)
			suite.Nil(err)

			response, err := suite.service.GetListeningEndpoints(suite.hasReadCtx, c.request)
			suite.NoError(err)

			suite.Equal(c.expectedPlopCount, len(response.ListeningEndpoints))
		})
	}

}

// Probably remove this before merging as it is redundant with the above
func (suite *PLOPServiceTestSuite) TestPLOPHappyPath() {
	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	request := &v1.GetProcessesListeningOnPortsRequest{
		DeploymentId: fixtureconsts.Deployment3,
	}
	// Check a deployment that doesn't exist
	response, err := suite.service.GetListeningEndpoints(suite.hasReadCtx, request)
	suite.NoError(err)

	newPlops := response.ListeningEndpoints
	suite.Len(newPlops, 0)

	request = &v1.GetProcessesListeningOnPortsRequest{
		DeploymentId: fixtureconsts.Deployment1,
	}
	// Check a deployment that doesn't exist
	response, err = suite.service.GetListeningEndpoints(suite.hasReadCtx, request)
	suite.NoError(err)

	newPlops = response.ListeningEndpoints
	suite.Len(newPlops, 1)

	protoassert.Equal(suite.T(), newPlops[0], &storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     fixtureconsts.Namespace1,
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		Signal: &storage.ProcessSignal{
			Name:         "test_process1",
			Args:         "test_arguments1",
			ExecFilePath: "test_path1",
		},
	})

}
