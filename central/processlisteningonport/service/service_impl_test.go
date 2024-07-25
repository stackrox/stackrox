//go:build sql_integration

package service

import (
	"context"
	"testing"
	"time"

	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

func TestPLOPService(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite
	datastore          datastore.DataStore
	service            Service
	store              plopStore.Store
	indicatorDataStore processIndicatorDataStore.DataStore

	postgres *pgtest.TestPostgres

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	hasAllCtx   context.Context
}

func (suite *ServiceTestSuite) SetupSuite() {
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

func (suite *ServiceTestSuite) SetupTest() {
	suite.postgres = pgtest.ForT(suite.T())
	suite.store = postgresStore.NewFullStore(suite.postgres.DB)

	indicatorStorage := processIndicatorStorage.New(suite.postgres.DB)
	indicatorSearcher := processIndicatorSearch.New(indicatorStorage)

	suite.indicatorDataStore, _ = processIndicatorDataStore.New(
		indicatorStorage, suite.store, indicatorSearcher, nil)
	suite.datastore = datastore.New(suite.store, suite.indicatorDataStore, suite.postgres)

	suite.service = New(suite.datastore)
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.postgres.Teardown(suite.T())
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
	for _, indicator := range indicators {
		id.SetIndicatorID(indicator)
	}

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

func (suite *ServiceTestSuite) addDeployments() {
        deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
        suite.Nil(err)
        suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: fixtureconsts.Deployment1, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))
        suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: fixtureconsts.Deployment2, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))
}

// TestPLOPAdd: Happy path for ProcessListeningOnPort, one PLOP object is added
// with a correct process indicator reference and could be fetched later.
func (suite *ServiceTestSuite) TestPLOPAdd() {
        indicators := getIndicators()

        plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

        suite.addDeployments()

        // Prepare indicators for FK
        suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
                suite.hasWriteCtx, indicators...))

        // Add PLOP referencing those indicators
        suite.NoError(suite.datastore.AddProcessListeningOnPort(
                suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

        // Fetch inserted PLOP back
	req := v1.GetProcessesListeningOnPortsRequest{DeploymentId: fixtureconsts.Deployment1}
        newPlops, err := suite.service.GetListeningEndpoints(
                suite.hasReadCtx, &req)
        suite.NoError(err)

        suite.Len(newPlops.ListeningEndpoints, 1)
        protoassert.Equal(suite.T(), newPlops.ListeningEndpoints[0], &storage.ProcessListeningOnPort{
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
        // Check a deployment that doesn't exist
	req = v1.GetProcessesListeningOnPortsRequest{DeploymentId: fixtureconsts.Deployment3}
        newPlops, err = suite.service.GetListeningEndpoints(
                suite.hasReadCtx, &req)
        suite.NoError(err)

        suite.Len(newPlops.ListeningEndpoints, 0)

	var empty *v1.Empty
        plopCounts, err := suite.service.CountListeningEndpoints(suite.hasReadCtx, empty)
        suite.NoError(err)

        expectedPlopCounts := map[string]int32{
                fixtureconsts.Deployment1:        1,
                fixtureconsts.Deployment2:        0,
        }
        suite.Equal(expectedPlopCounts, plopCounts.Counts)
}

// TestPLOPServiceSAC: Test various access scopes
func (suite *ServiceTestSuite) TestPLOPServiceSAC() {
	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	suite.addDeployments()

        cases := map[string]struct {
                checker       sac.ScopeCheckerCore
                expectAllowed bool
		expectedPlopCounts map[string]int32
        }{
                "all access": {
                        checker:       sac.AllowAllAccessScopeChecker(),
                        expectAllowed: true,
                        expectedPlopCounts: map[string]int32{
                                fixtureconsts.Deployment1: 1,
                                fixtureconsts.Deployment2: 0,
                        },
                },
		"read and write access": {
                        checker: sac.AllowFixedScopes(
                                sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
                                sac.ResourceScopeKeys(resources.DeploymentExtension),
                                sac.ClusterScopeKeys(fixtureconsts.Cluster1),
                                sac.NamespaceScopeKeys(fixtureconsts.Namespace1),
                        ),
                        expectAllowed: true,
                        expectedPlopCounts: map[string]int32{
                                fixtureconsts.Deployment1: 1,
                                fixtureconsts.Deployment2: 0,
                        },
                },
                "access to wrong namespace": {
                        checker: sac.AllowFixedScopes(
                                sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
                                sac.ResourceScopeKeys(resources.DeploymentExtension),
                                sac.ClusterScopeKeys(fixtureconsts.Cluster1),
                                sac.NamespaceScopeKeys(fixtureconsts.Namespace2),
                        ),
                        expectAllowed:      false,
                        expectedPlopCounts: map[string]int32{},
                },
                "access to wrong cluster": {
                        checker: sac.AllowFixedScopes(
                                sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
                                sac.ResourceScopeKeys(resources.DeploymentExtension),
                                sac.ClusterScopeKeys(fixtureconsts.Cluster2),
                                sac.NamespaceScopeKeys(fixtureconsts.Namespace1),
                        ),
                        expectAllowed:      false,
                        expectedPlopCounts: map[string]int32{},
                },
		                "cluster level access": {
                        checker: sac.AllowFixedScopes(
                                sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
                                sac.ResourceScopeKeys(resources.DeploymentExtension),
                                sac.ClusterScopeKeys(fixtureconsts.Cluster1),
                        ),
                        expectAllowed: true,
                        expectedPlopCounts: map[string]int32{
                                fixtureconsts.Deployment1: 1,
                                fixtureconsts.Deployment2: 0,
                        },
                },
		"no access": {
                        checker:       sac.DenyAllAccessScopeChecker(),
			expectAllowed:	false,
                        expectedPlopCounts: map[string]int32{},
		},
	}

	req := v1.GetProcessesListeningOnPortsRequest{DeploymentId: fixtureconsts.Deployment1}

	for name, c := range cases {
		suite.Run(name, func() {
			ctx := sac.WithGlobalAccessScopeChecker(context.Background(), c.checker)
			processListeningOnPortsResponse, err := suite.service.GetListeningEndpoints(ctx, &req)
			var empty *v1.Empty
			plopCounts, countErr := suite.service.CountListeningEndpoints(ctx, empty)
			suite.Equal(c.expectedPlopCounts, plopCounts.Counts)

			if c.expectAllowed {
				suite.NoError(err)
				suite.NoError(countErr)
				suite.Len(processListeningOnPortsResponse.ListeningEndpoints, 1)
                                protoassert.Equal(suite.T(), processListeningOnPortsResponse.ListeningEndpoints[0], &storage.ProcessListeningOnPort{
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
			} else {
				suite.ErrorIs(err, sac.ErrResourceAccessDenied)
			}

		})
        }
}
