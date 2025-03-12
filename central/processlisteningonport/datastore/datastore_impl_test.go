//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestPLOPDataStore(t *testing.T) {
	suite.Run(t, new(PLOPDataStoreTestSuite))
}

type PLOPDataStoreTestSuite struct {
	suite.Suite
	datastore          DataStore
	store              plopStore.Store
	indicatorDataStore processIndicatorDataStore.DataStore

	postgres *pgtest.TestPostgres

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context
	hasAllCtx   context.Context
}

func (suite *PLOPDataStoreTestSuite) SetupSuite() {
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

func (suite *PLOPDataStoreTestSuite) SetupTest() {
	suite.postgres = pgtest.ForT(suite.T())
	suite.store = postgresStore.NewFullStore(suite.postgres.DB)

	indicatorStorage := processIndicatorStorage.New(suite.postgres.DB)
	indicatorSearcher := processIndicatorSearch.New(indicatorStorage)

	suite.indicatorDataStore, _ = processIndicatorDataStore.New(
		indicatorStorage, suite.store, indicatorSearcher, nil)
	suite.datastore = New(suite.store, suite.indicatorDataStore, suite.postgres)
}

func (suite *PLOPDataStoreTestSuite) TearDownTest() {
	suite.postgres.Teardown(suite.T())
}

func (suite *PLOPDataStoreTestSuite) getPlopsFromDB() []*storage.ProcessListeningOnPortStorage {
	plopsFromDB := []*storage.ProcessListeningOnPortStorage{}
	err := suite.datastore.WalkAll(suite.hasReadCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			plopsFromDB = append(plopsFromDB, plop)
			return nil
		})

	suite.NoError(err)

	return plopsFromDB
}

func (suite *PLOPDataStoreTestSuite) getProcessIndicatorsFromDB() []*storage.ProcessIndicator {
	indicatorsFromDB := []*storage.ProcessIndicator{}
	err := suite.indicatorDataStore.WalkAll(suite.hasWriteCtx,
		func(processIndicator *storage.ProcessIndicator) error {
			indicatorsFromDB = append(indicatorsFromDB, processIndicator)
			return nil
		})

	suite.NoError(err)

	return indicatorsFromDB
}

func getPlopMap(plops []*storage.ProcessListeningOnPortStorage) map[string]*storage.ProcessListeningOnPortStorage {
	plopMap := make(map[string]*storage.ProcessListeningOnPortStorage)

	for _, plop := range plops {
		plopMap[getPlopKey(plop)] = plop
	}

	return plopMap
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

func (suite *PLOPDataStoreTestSuite) addDeployments() {
	deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
	suite.Nil(err)
	suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: fixtureconsts.Deployment1, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))
	suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: fixtureconsts.Deployment2, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))
}

// TestPLOPAdd: Happy path for ProcessListeningOnPort, one PLOP object is added
// with a correct process indicator reference and could be fetched later.
func (suite *PLOPDataStoreTestSuite) TestPLOPAdd() {
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
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

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

	// Check a deployment that doesn't exist
	newPlops, err = suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment3)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that newly added PLOP object doesn't have Process field set in
	// the serialized column (because all the info is stored in the referenced
	// process indicator record)
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       plopObjects[0].GetDeploymentId(),
		PodUid:             plopObjects[0].GetPodUid(),
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddNoDeployments: Add PLOPs without a matching deployment in the deployments table
func (suite *PLOPDataStoreTestSuite) TestPLOPAddNoDeployments() {
	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that newly added PLOP object doesn't have Process field set in
	// the serialized column (because all the info is stored in the referenced
	// process indicator record)
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       plopObjects[0].GetDeploymentId(),
		PodUid:             plopObjects[0].GetPodUid(),
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPSAC: Tests getting the PLOPs with various levels of access
func (suite *PLOPDataStoreTestSuite) TestPLOPSAC() {
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
		ctx           context.Context
		expectAllowed bool
	}{
		"all access": {
			ctx:           sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker()),
			expectAllowed: true,
		},
		"access to cluster and namespace": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(fixtureconsts.Cluster1),
				sac.NamespaceScopeKeys(fixtureconsts.Namespace1),
			)),
			expectAllowed: true,
		},
		"read and write access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(fixtureconsts.Cluster1),
				sac.NamespaceScopeKeys(fixtureconsts.Namespace1),
			)),
			expectAllowed: true,
		},
		"access to wrong namespace": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(fixtureconsts.Cluster1),
				sac.NamespaceScopeKeys(fixtureconsts.Namespace2),
			)),
			expectAllowed: false,
		},
		"access to wrong cluster": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(fixtureconsts.Cluster2),
				sac.NamespaceScopeKeys(fixtureconsts.Namespace1),
			)),
			expectAllowed: false,
		},
		"cluster level access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.DeploymentExtension),
				sac.ClusterScopeKeys(fixtureconsts.Cluster1),
			)),
			expectAllowed: true,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			newPlops, err := suite.datastore.GetProcessListeningOnPort(
				c.ctx, fixtureconsts.Deployment1)
			if c.expectAllowed {
				suite.NoError(err)
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

			} else {
				suite.ErrorIs(err, sac.ErrResourceAccessDenied)
				suite.Len(newPlops, 0)
			}
		})
	}

	// Verify that newly added PLOP object doesn't have Process field set in
	// the serialized column (because all the info is stored in the referenced
	// process indicator record)
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       plopObjects[0].GetDeploymentId(),
		PodUid:             plopObjects[0].GetPodUid(),
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosed: Happy path for ProcessListeningOnPort closing, one PLOP object is added
// with a correct process indicator reference and CloseTimestamp set. It will
// be exluded from the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosed() {

	indicators := getIndicators()

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsActive...))

	// Close PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsClosed...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's closed and excluded from the API response
	suite.Len(newPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjectsClosed[0].GetPort(),
		Protocol:           plopObjectsClosed[0].GetProtocol(),
		CloseTimestamp:     plopObjectsClosed[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             plopObjectsClosed[0].GetPodUid(),
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenTwice: Add the same open PLOP twice
// There should only be one PLOP in the storage and one
// PLOP returned by the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenTwice() {
	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP again
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

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

	// Verify that newly added PLOP object doesn't have Process field set in
	// the serialized column (because all the info is stored in the referenced
	// process indicator record)
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddCloseTwice: Add the same closed PLOP twice
// There should only be one PLOP in the storage
func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseTwice() {
	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP again
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that newly added PLOP object doesn't have Process field set in
	// the serialized column (because all the info is stored in the referenced
	// process indicator record)
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPReopen: One PLOP object is added with a correct process indicator
// reference and CloseTimestamp set to nil. It will reopen an existing PLOP and
// present in the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPReopen() {
	indicators := getIndicators()

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsActive...))

	// Close PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsClosed...))

	// Reopen PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsActive...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// The PLOP is reported since it is in the open state
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

	// Verify that PLOP object was updated and no new records were created
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjectsActive[0].GetPort(),
		Protocol:           plopObjectsActive[0].GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPCloseSameTimestamp: One PLOP object is added with a correct process
// indicator reference and CloseTimestamp set to the same as existing one.
func (suite *PLOPDataStoreTestSuite) TestPLOPCloseSameTimestamp() {

	indicators := getIndicators()

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsActive...))

	// Close PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsClosed...))

	// Send same close event again
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjectsClosed...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's closed and excluded from the API response
	suite.Len(newPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjectsClosed[0].GetPort(),
		Protocol:           plopObjectsClosed[0].GetProtocol(),
		CloseTimestamp:     plopObjectsClosed[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosedSameBatch: One PLOP object is added with a correct process
// indicator reference with and without CloseTimestamp set in the same batch.
// This will excercise logic of batch normalization. It will be exluded from
// the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedSameBatch() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject, &closedPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's closed and excluded from the API response
	suite.Len(newPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[1].GetPort(),
		Protocol:           plopObjects[1].GetProtocol(),
		CloseTimestamp:     plopObjects[1].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosedWithoutActive: one PLOP object is added with a correct
// process indicator reference and CloseTimestamp set, without having
// previously active PLOP. Will be stored in the db as closed and excluded from
// the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedWithoutActive() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Confirm that the database is empty before anything is inserted into it
	plopsFromDB := suite.getPlopsFromDB()
	suite.Len(plopsFromDB, 0)

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddNoIndicator: A PLOP object with a wrong process indicator
// reference. It's being stored in the database and will be returned by
// the API even though it is not matched to a process.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddNoIndicator() {

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Verify that the plop table is empty before the test
	plopsFromDB := []*storage.ProcessListeningOnPortStorage{}
	err := suite.datastore.WalkAll(suite.hasWriteCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			plopsFromDB = append(plopsFromDB, plop)
			return nil
		})
	suite.NoError(err)
	suite.Len(plopsFromDB, 0)

	// Verify that the process indicator table is empty before the test
	indicatorsFromDB := suite.getProcessIndicatorsFromDB()
	suite.Len(indicatorsFromDB, 0)

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 1)
	protoassert.Equal(suite.T(), newPlops[0], &storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     fixtureconsts.Namespace1,
		DeploymentId:  fixtureconsts.Deployment1,
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

	// Verify the state of the table after the test
	// Process should not be nil as we were not able to find
	// a matching process indicator
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: getIndicators()[0].GetId(),
		Closed:             false,
		Process:            plopObjects[0].GetProcess(),
		DeploymentId:       plopObjects[0].GetDeploymentId(),
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosedNoIndicator: A PLOP object with a wrong process indicator
// reference and CloseTimestamp set. It's stored in the database, but
// as it is closed it will not be returned by the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedNoIndicator() {

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that newly added PLOP has Process field set, because we were not
	// able to establish reference to a process indicator and don't want to
	// loose the data
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: getIndicators()[0].GetId(),
		Closed:             true,
		Process:            plopObjects[0].GetProcess(),
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenNoIndicatorThenClose Adds an open PLOP object with no matching
// indicator. Adds an indicator and then closes the PLOP
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenNoIndicatorThenClose() {

	openPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, openPlopObjects...))

	indicators := getIndicators()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Add closed PLOP now with a matching indicator
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, closedPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that newly added PLOP has Process field set, because we were not
	// able to establish reference to a process indicator and don't want to
	// loose the data
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               closedPlopObjects[0].GetPort(),
		Protocol:           closedPlopObjects[0].GetProtocol(),
		CloseTimestamp:     closedPlopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: getIndicators()[0].GetId(),
		Closed:             true,
		Process:            closedPlopObjects[0].GetProcess(),
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenAndClosedNoIndicator: An open PLOP object with no matching
// process indicator is sent. Then the PLOP object is closed. We expect the
// database to contain one PLOP and the database to return nothing, because
// it is closed.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenAndClosedNoIndicator() {
	openPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}
	closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	suite.addDeployments()

	// Add open PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, openPlopObjects...))

	// Add closed PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, closedPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that newly added PLOP has Process field set, because we were not
	// able to establish reference to a process indicator and don't want to
	// loose the data
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               closedPlopObjects[0].GetPort(),
		Protocol:           closedPlopObjects[0].GetProtocol(),
		CloseTimestamp:     closedPlopObjects[0].GetCloseTimestamp(),
		ProcessIndicatorId: getIndicators()[0].GetId(),
		Closed:             true,
		Process:            closedPlopObjects[0].GetProcess(),
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddMultipleIndicators: A PLOP object is added with a valid reference
// that matches one of two process indicators
func (suite *PLOPDataStoreTestSuite) TestPLOPAddMultipleIndicators() {
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
			ContainerName: "test_container1",
			Namespace:     fixtureconsts.Namespace1,

			Signal: &storage.ProcessSignal{
				Name:         "test_process1",
				Args:         "test_arguments1",
				ExecFilePath: "test_path1",
			},
		},
	}

	for _, indicator := range indicators {
		id.SetIndicatorID(indicator)
	}
	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

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

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopObjects[0].GetPort(),
		Protocol:           plopObjects[0].GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenThenCloseAndOpenSameBatch Sends an open PLOP with a matching indicator.
// Then closes and opens the PLOP object in the same batch.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenThenCloseAndOpenSameBatch() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&closedPlopObject,
		&openPlopObject,
	}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// The plop is opened. Then in the batch it is closed and opened, so it is in
	// its original open state.
	suite.Len(newPlops, 1)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               openPlopObject.GetPort(),
		Protocol:           openPlopObject.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddCloseThenCloseAndOpenSameBatch Adds a closed PLOP object with an indicator.
// Then in the next batch the PLOP object is opened and closed.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseThenCloseAndOpenSameBatch() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&openPlopObject,
		&closedPlopObject,
	}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's closed and excluded from the API response
	suite.Len(newPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               closedPlopObject.GetPort(),
		Protocol:           closedPlopObject.GetProtocol(),
		CloseTimestamp:     closedPlopObject.GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddCloseBatchOutOfOrderMoreClosed: Excersice batching logic when
// having more "closed" PLOP events
func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseBatchOutOfOrderMoreClosed() {

	indicators := getIndicators()

	time1 := time.Now()
	time2 := time.Now().Local().Add(time.Hour * time.Duration(1))
	time3 := time.Now().Local().Add(time.Hour * time.Duration(2))

	closedPlopObject1 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time1),
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

	closedPlopObject2 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time2),
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

	closedPlopObject3 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time3),
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject1}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&closedPlopObject3,
		&openPlopObject,
		&closedPlopObject2,
		&openPlopObject,
	}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP in an open and closed state multiple times
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's closed and excluded from the API response
	suite.Len(newPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               closedPlopObject3.GetPort(),
		Protocol:           closedPlopObject3.GetProtocol(),
		CloseTimestamp:     closedPlopObject3.GetCloseTimestamp(),
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddCloseBatchOutOfOrderMoreOpen: Excersice batching logic when
// having more "open" PLOP events
func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseBatchOutOfOrderMoreOpen() {

	indicators := getIndicators()

	time1 := time.Now()
	time2 := time.Now().Local().Add(time.Hour * time.Duration(1))
	time3 := time.Now().Local().Add(time.Hour * time.Duration(2))

	closedPlopObject1 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time1),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodUid:       fixtureconsts.PodUID1,
	}

	closedPlopObject2 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time2),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodUid:       fixtureconsts.PodUID1,
	}

	closedPlopObject3 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time3),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodUid:       fixtureconsts.PodUID1,
	}

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject1}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&openPlopObject,
		&closedPlopObject3,
		&openPlopObject,
		&closedPlopObject2,
		&openPlopObject,
	}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's open and included into the API response
	suite.Len(newPlops, 1)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               closedPlopObject3.GetPort(),
		Protocol:           closedPlopObject3.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPDeleteAndCreateDeployment: Creates a deployment with PLOP and
// matching process indicator. Closes the PLOP and deletes the process indicator.
// Creates the PLOP with a new DeploymentId and matching process indicator.
func (suite *PLOPDataStoreTestSuite) TestPLOPDeleteAndCreateDeployment() {
	initialIndicators := []*storage.ProcessIndicator{
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
	}

	id.SetIndicatorID(initialIndicators[0])

	openPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, initialIndicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, openPlopObjects...))

	closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Add the same PLOP in a closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, closedPlopObjects...))

	idsToDelete := []string{initialIndicators[0].Id}

	// Verify the state of the PLOP table after opening and closing the endpoint
	plopsFromDB1 := suite.getPlopsFromDB()
	suite.Len(plopsFromDB1, 1)

	expectedPlopStorage1 := &storage.ProcessListeningOnPortStorage{
		Id:                 plopsFromDB1[0].GetId(),
		Port:               openPlopObject.GetPort(),
		Protocol:           openPlopObject.GetProtocol(),
		CloseTimestamp:     closedPlopObject.GetCloseTimestamp(),
		ProcessIndicatorId: initialIndicators[0].GetId(),
		Closed:             true,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage1, plopsFromDB1[0])

	// Delete the indicator
	suite.NoError(suite.indicatorDataStore.RemoveProcessIndicators(
		suite.hasWriteCtx, idsToDelete))

	_, err := suite.datastore.RemovePLOPsWithoutProcessIndicatorOrProcessInfo(suite.hasWriteCtx)
	suite.NoError(err)

	// Verify the state of the PLOP table after deleting the process indicator
	plopsFromDB2 := suite.getPlopsFromDB()
	suite.Len(plopsFromDB2, 0)

	// Create a new indicator in a new deployment
	newIndicators := []*storage.ProcessIndicator{
		{
			Id:           fixtureconsts.ProcessIndicatorID1,
			DeploymentId: fixtureconsts.Deployment2,
			// Keeping the same PodId even though a new deployment almost certainly would have a new PodId
			// The code is robust even for that case
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
	}

	id.SetIndicatorID(newIndicators[0])

	// Set the PLOP to the new DeploymentId
	newOpenPlopObject := storage.ProcessListeningOnPortFromSensor{
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
		DeploymentId: fixtureconsts.Deployment2,
		PodUid:       fixtureconsts.PodUID1,
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
	}

	newOpenPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&newOpenPlopObject}

	// Add new indicator
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, newIndicators...))

	// Add the PLOP with the new DeploymentId
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, newOpenPlopObjects...))

	// Fetch inserted PLOP back from the new deployment
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment2)
	suite.NoError(err)

	// It's open and included in the API response for the new deployment
	suite.Len(newPlops, 1)
	protoassert.Equal(suite.T(), newPlops[0], &storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment2,
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

	// Fetch inserted PLOP back from the old deployment
	oldPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's closed and doesn't appear in the API
	suite.Len(oldPlops, 0)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               openPlopObject.GetPort(),
		Protocol:           openPlopObject.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: newIndicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment2,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPNoProcessInformation: PLOP should not appear in the API if there is no process information
func (suite *PLOPDataStoreTestSuite) TestPLOPNoProcessInformation() {
	indicators := getIndicators()

	plopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 indicators[0].GetId(), // Id doesn't matter here. Just needs to be the right type
		Port:               1234,
		Protocol:           storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	suite.addDeployments()

	// It is not possible to add a PLOP from sensor with no process info
	// so upsert directly to the database. In the tests when a process indicator
	// is deleted the PLOP is also deleted from the database. This does not seem
	// to be the case in reality.
	suite.NoError(suite.store.Upsert(
		suite.hasWriteCtx, plopStorage))

	// Fetch inserted PLOP back
	// It doesn't appear in the API, because it has no process info
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that the PLOP is in the database
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	protoassert.Equal(suite.T(), plopStorage, newPlopsFromDB[0])
}

// TestRemovePlopsByPod: Create two plops and remove one of them by PodUID
func (suite *PLOPDataStoreTestSuite) TestRemovePlopsByPod() {

	plop1 := storage.ProcessListeningOnPortFromSensor{
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

	plop2 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName2,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		PodUid:       fixtureconsts.PodUID2,
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
	}

	suite.addDeployments()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&plop1, &plop2}

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Verify the newly added PLOP objects before deleting one of the pods
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 2)

	id1 := id.GetIndicatorIDFromProcessIndicatorUniqueKey(plop1.Process)
	id2 := id.GetIndicatorIDFromProcessIndicatorUniqueKey(plop2.Process)

	plopMap := getPlopMap(newPlopsFromDB)

	expectedPlopStorage := []*storage.ProcessListeningOnPortStorage{
		{
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
			ProcessIndicatorId: id1,
			Closed:             false,
			Process:            plopObjects[0].GetProcess(),
			DeploymentId:       plopObjects[0].GetDeploymentId(),
			PodUid:             plopObjects[0].GetPodUid(),
			ClusterId:          fixtureconsts.Cluster1,
			Namespace:          fixtureconsts.Namespace1,
		},
		{
			Port:               plopObjects[1].GetPort(),
			Protocol:           plopObjects[1].GetProtocol(),
			CloseTimestamp:     plopObjects[1].GetCloseTimestamp(),
			ProcessIndicatorId: id2,
			Closed:             false,
			Process:            plopObjects[1].GetProcess(),
			DeploymentId:       plopObjects[1].GetDeploymentId(),
			PodUid:             plopObjects[1].GetPodUid(),
			ClusterId:          fixtureconsts.Cluster1,
			Namespace:          fixtureconsts.Namespace1,
		},
	}

	expectedPlopStorageMap := getPlopMap(expectedPlopStorage)

	for key, expectedPlop := range expectedPlopStorageMap {
		// We cannot know the Id in advance so set it here.
		expectedPlop.Id = plopMap[key].Id
		protoassert.Equal(suite.T(), expectedPlop, plopMap[key])
	}

	// Remove the PLOP for the pod
	suite.NoError(suite.datastore.RemovePlopsByPod(
		suite.hasWriteCtx, fixtureconsts.PodUID1))

	// Verify the PLOP has been deleted for the specified pod
	newPlopsFromDB2 := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB2, 1)

	expectedPlopStorage1 := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB2[0].GetId(),
		Port:               plopObjects[1].GetPort(),
		Protocol:           plopObjects[1].GetProtocol(),
		CloseTimestamp:     plopObjects[1].GetCloseTimestamp(),
		ProcessIndicatorId: id2,
		Closed:             false,
		Process:            plopObjects[1].GetProcess(),
		DeploymentId:       plopObjects[1].GetDeploymentId(),
		PodUid:             plopObjects[1].GetPodUid(),
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage1, newPlopsFromDB2[0])

}

// TestPLOPUpdatePodUidFromBlank Add a PLOP without a PodUid and then
// the same PLOP is added with a PodUid
func (suite *PLOPDataStoreTestSuite) TestPLOPUpdatePodUidFromBlank() {
	indicators := getIndicators()

	plopWithoutPodUID := storage.ProcessListeningOnPortFromSensor{
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
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
	}

	plopWithPodUID := storage.ProcessListeningOnPortFromSensor{
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&plopWithoutPodUID}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 1)
	protoassert.Equal(suite.T(), newPlops[0], &storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
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

	// Verify the state of the table
	newPlopsFromDB := suite.getPlopsFromDB()

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopWithoutPodUID.GetPort(),
		Protocol:           plopWithoutPodUID.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])

	plopObjects = []*storage.ProcessListeningOnPortFromSensor{&plopWithPodUID}

	// Add PLOP with PodUid
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err = suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

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

	// Verify the state of the table
	newPlopsFromDB = suite.getPlopsFromDB()

	expectedPlopStorage = &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopWithoutPodUID.GetPort(),
		Protocol:           plopWithoutPodUID.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPUpdatePodUidFromBlankClosed Add a closed PLOP without a PodUid and then
// the same PLOP is added with a PodUid
func (suite *PLOPDataStoreTestSuite) TestPLOPUpdatePodUidFromBlankClosed() {
	indicators := getIndicators()

	time1 := time.Now()
	time2 := time.Now().Local().Add(time.Hour * time.Duration(1))

	plopWithoutPodUID := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time1),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodName1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
		DeploymentId: fixtureconsts.Deployment1,
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
	}

	plopWithPodUID := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time2),
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&plopWithoutPodUID}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	plopObjects = []*storage.ProcessListeningOnPortFromSensor{&plopWithPodUID}

	// Add PLOP with PodUid
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err = suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	id1 := id.GetIndicatorIDFromProcessIndicatorUniqueKey(plopWithPodUID.Process)

	expectedPlopStorage := []*storage.ProcessListeningOnPortStorage{
		{
			Id:                 newPlopsFromDB[0].Id,
			Port:               plopObjects[0].GetPort(),
			Protocol:           plopObjects[0].GetProtocol(),
			CloseTimestamp:     plopObjects[0].GetCloseTimestamp(),
			ProcessIndicatorId: id1,
			Closed:             true,
			DeploymentId:       plopObjects[0].GetDeploymentId(),
			PodUid:             plopObjects[0].GetPodUid(),
			ClusterId:          fixtureconsts.Cluster1,
			Namespace:          fixtureconsts.Namespace1,
		},
	}

	protoassert.SlicesEqual(suite.T(), expectedPlopStorage, newPlopsFromDB)
}

// TestPLOPAddOpenThenCloseAndOpenSameBatchWithPodUid Sends an open PLOP with a matching indicator.
// Then closes and opens the PLOP object in the same batch. The first PLOP does not have a PodUid
// and then in the batch with two PLOP events they both have the same PodUid.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenThenCloseAndOpenSameBatchWithPodUid() {

	indicators := getIndicators()

	openPlopObjectWithoutPodUID := storage.ProcessListeningOnPortFromSensor{
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
		Namespace:    fixtureconsts.Namespace1,
	}

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObjectWithoutPodUID}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&closedPlopObject,
		&openPlopObject,
	}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// The plop is opened. Then in the batch it is closed and opened, so it is in
	// its original open state.
	suite.Len(newPlops, 1)

	// Verify the state of the table after the test
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               openPlopObject.GetPort(),
		Protocol:           openPlopObject.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPUpdateClusterIdFromBlank Add a PLOP without a ClusterId and then
// the same PLOP is added with a ClusterId
func (suite *PLOPDataStoreTestSuite) TestPLOPUpdateClusterIdFromBlank() {
	indicators := getIndicators()

	plopWithoutClusterID := storage.ProcessListeningOnPortFromSensor{
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
		Namespace:    fixtureconsts.Namespace1,
	}

	plopWithClusterID := storage.ProcessListeningOnPortFromSensor{
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&plopWithoutClusterID}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, "", plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

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

	// Verify the state of the table
	newPlopsFromDB := suite.getPlopsFromDB()

	expectedPlopStorage := &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopWithoutClusterID.GetPort(),
		Protocol:           plopWithoutClusterID.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])

	plopObjects = []*storage.ProcessListeningOnPortFromSensor{&plopWithClusterID}

	// Add PLOP with PodUid
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err = suite.datastore.GetProcessListeningOnPort(
		suite.hasReadCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

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

	// Verify the state of the table
	newPlopsFromDB = suite.getPlopsFromDB()

	expectedPlopStorage = &storage.ProcessListeningOnPortStorage{
		Id:                 newPlopsFromDB[0].GetId(),
		Port:               plopWithoutClusterID.GetPort(),
		Protocol:           plopWithoutClusterID.GetProtocol(),
		CloseTimestamp:     nil,
		ProcessIndicatorId: indicators[0].GetId(),
		Closed:             false,
		Process:            nil,
		DeploymentId:       fixtureconsts.Deployment1,
		PodUid:             fixtureconsts.PodUID1,
		ClusterId:          fixtureconsts.Cluster1,
		Namespace:          fixtureconsts.Namespace1,
	}

	protoassert.Equal(suite.T(), expectedPlopStorage, newPlopsFromDB[0])
}

func makeRandomString(length int) string {
	var charset = []byte("asdfqwert")
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomString)
}

func (suite *PLOPDataStoreTestSuite) makeRandomPlops(nport int, nprocess int, npod int, deployment string) []*storage.ProcessListeningOnPortFromSensor {
	count := 0

	nplops := 2 * nprocess * npod * nport

	plops := make([]*storage.ProcessListeningOnPortFromSensor, nplops)
	for podIdx := 0; podIdx < npod; podIdx++ {
		podID := makeRandomString(10)
		podUid := uuid.NewV4().String()
		for processIdx := 0; processIdx < nprocess; processIdx++ {
			execFilePath := makeRandomString(10)
			for port := 0; port < nport; port++ {

				plopTCP := &storage.ProcessListeningOnPortFromSensor{
					Port:           uint32(port),
					Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
					CloseTimestamp: nil,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               podID,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: execFilePath,
					},
					DeploymentId: deployment,
					ClusterId:    fixtureconsts.Cluster1,
					PodUid:       podUid,
				}
				plopUDP := &storage.ProcessListeningOnPortFromSensor{
					Port:           uint32(port),
					Protocol:       storage.L4Protocol_L4_PROTOCOL_UDP,
					CloseTimestamp: nil,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               podID,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: execFilePath,
					},
					DeploymentId: deployment,
					ClusterId:    fixtureconsts.Cluster1,
					PodUid:       podUid,
				}
				plops[count] = plopTCP
				count++
				plops[count] = plopUDP
				count++
			}
		}
	}
	return plops
}

// TestDeletePods: The purpose of this test is to check for a race condition between RemovePlopsByPod
// and AddProcessListeningOnPort. They should not delete PLOPs simultaneously.
func (suite *PLOPDataStoreTestSuite) TestDeletePods() {
	nport := 30
	nprocess := 30
	npod := 30

	plopObjects := suite.makeRandomPlops(nport, nprocess, npod, fixtureconsts.Deployment1)

	// Get a set of PodUids so that we can delete by PodUid later
	podUids := set.NewStringSet()
	for _, plop := range plopObjects {
		podUids.Add(plop.PodUid)
	}

	// Add the PLOPs
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	// Close the PLOPs
	for _, plop := range plopObjects {
		plop.CloseTimestamp = protoconv.ConvertTimeToTimestamp(time.Now())
	}

	// Add the closed PLOPs. The PLOPs are opened and then closed.
	// The reason for this is that we need to have UpsertMany delete PLOPs.
	// This can only happen if there are updates for existing PLOPs.
	// It is done async so that the UpsertMany in AddProcessListeningOnPort
	// runs at the same time as PLOPs are deleted by pod
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.NoError(suite.datastore.AddProcessListeningOnPort(
			suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))
	}()

	// The pods are deleted and all PLOPs are deleted by pod
	for podUID := range podUids {
		suite.NoError(suite.datastore.RemovePlopsByPod(suite.hasWriteCtx, podUID))
		// Sleep a little bit to increase the chance that the PLOPs will be deleted by RemovePlopsByPod at
		// the same time as they are being deleted by the call to UpsertMany in AddProcessListeningOnPort.
		// Without the sleeps this loop will finish before UpsertMany is reached in AddProcessListeningOnPort.
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()
}

func (suite *PLOPDataStoreTestSuite) TestRemoveOrphanedPLOPs() {
	orphanWindow := 30 * time.Minute

	cases := []struct {
		name                  string
		initialPlops          []*storage.ProcessListeningOnPortStorage
		deployments           set.FrozenStringSet
		pods                  set.FrozenStringSet
		expectedPlopDeletions []string
	}{
		{
			name: "no deployments nor pods - remove plops since there are no pods",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
				fixtures.GetPlopStorage4(),
				fixtures.GetPlopStorage5(),
				fixtures.GetPlopStorage6(),
			},
			deployments:           set.NewFrozenStringSet(),
			pods:                  set.NewFrozenStringSet(),
			expectedPlopDeletions: []string{fixtureconsts.PlopUID1, fixtureconsts.PlopUID2, fixtureconsts.PlopUID3, fixtureconsts.PlopUID4, fixtureconsts.PlopUID5, fixtureconsts.PlopUID6},
		},
		{
			name: "deployments one missing pod - remove plops with PodUid with no matching pod even though there are deployments",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
				fixtures.GetPlopStorage4(),
				fixtures.GetPlopStorage5(),
				fixtures.GetPlopStorage6(),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment6, fixtureconsts.Deployment5, fixtureconsts.Deployment3),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2),
			expectedPlopDeletions: []string{fixtureconsts.PlopUID6},
		},
		{
			name: "one missing deployments no missing pods - remove plops with no matching deployments even though there are matching pods",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
				fixtures.GetPlopStorage4(),
				fixtures.GetPlopStorage5(),
				fixtures.GetPlopStorage6(),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment5, fixtureconsts.Deployment3),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedPlopDeletions: []string{fixtureconsts.PlopUID1, fixtureconsts.PlopUID4},
		},
		{
			name: "no missing deployments or pods but plops are expired - remove all expired plops",
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage4(),
				fixtures.GetPlopStorageExpired1(),
				fixtures.GetPlopStorageExpired2(),
				fixtures.GetPlopStorageExpired3(),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment6, fixtureconsts.Deployment5, fixtureconsts.Deployment3),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedPlopDeletions: []string{fixtureconsts.PlopUID7, fixtureconsts.PlopUID8, fixtureconsts.PlopUID9},
		},
	}
	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			suite.TearDownTest()
			suite.SetupTest()
			// Add deployments if necessary
			deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
			suite.Nil(err)
			for _, deploymentID := range c.deployments.AsSlice() {
				suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: deploymentID, ClusterId: fixtureconsts.Cluster1}))
			}

			for _, podID := range c.pods.AsSlice() {
				insertPod := fmt.Sprintf("INSERT INTO pods (id, clusterid) VALUES ('%s', '%s')", podID, fixtureconsts.Cluster1)
				_, err := suite.postgres.DB.Exec(suite.hasWriteCtx, insertPod)
				suite.Nil(err)
			}

			err = suite.store.UpsertMany(suite.hasWriteCtx, c.initialPlops)
			suite.NoError(err)
			plopCount, err := suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
			suite.NoError(err)
			suite.Equal(len(c.initialPlops), plopCount)

			suite.datastore.PruneOrphanedPLOPs(suite.hasWriteCtx, orphanWindow)

			plopCount, err = suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
			suite.NoError(err)
			suite.Equal(len(c.initialPlops)-len(c.expectedPlopDeletions), plopCount)

			ids, err := suite.store.GetIDs(suite.hasReadCtx)
			suite.NoError(err)
			for id := range ids {
				suite.NotContains(c.expectedPlopDeletions, id)
			}

		})
	}
}

func newIndicatorWithDeployment(id string, age time.Duration, deploymentID string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            id,
		DeploymentId:  deploymentID,
		ContainerName: "",
		PodId:         "",
		Signal: &storage.ProcessSignal{
			Time: protoconv.NowMinus(age),
		},
	}
}

func newIndicatorWithDeploymentAndPod(id string, age time.Duration, deploymentID, podUID string) *storage.ProcessIndicator {
	indicator := newIndicatorWithDeployment(id, age, deploymentID)
	indicator.PodUid = podUID
	return indicator
}

func (suite *PLOPDataStoreTestSuite) TestRemoveOrphanedPLOPsByProcesses() {
	orphanWindow := 30 * time.Minute

	cases := []struct {
		name                  string
		initialProcesses      []*storage.ProcessIndicator
		initialPlops          []*storage.ProcessListeningOnPortStorage
		deployments           set.FrozenStringSet
		pods                  set.FrozenStringSet
		expectedPlopDeletions []string
	}{
		{
			name: "no deployments nor pods - remove all old indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
				fixtures.GetPlopStorage4(),
				fixtures.GetPlopStorage5(),
				fixtures.GetPlopStorage6(),
			},
			deployments: set.NewFrozenStringSet(),
			pods:        set.NewFrozenStringSet(),
			expectedPlopDeletions: []string{
				fixtureconsts.PlopUID1,
				fixtureconsts.PlopUID2,
				fixtureconsts.PlopUID3,
				fixtureconsts.PlopUID4,
				fixtureconsts.PlopUID5,
				fixtureconsts.PlopUID6,
			},
		},
		{
			name: "no deployments nor pods - remove no new orphaned indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 20*time.Minute, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 20*time.Minute, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
			},
			deployments:           set.NewFrozenStringSet(),
			pods:                  set.NewFrozenStringSet(),
			expectedPlopDeletions: nil,
		},
		{
			name: "all pods separate deployments - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment6, fixtureconsts.Deployment5, fixtureconsts.Deployment3),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedPlopDeletions: nil,
		},
		{
			name: "all pods same deployment - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID3),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment6),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedPlopDeletions: nil,
		},
		{
			name: "some pods separate deployments - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment3),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID3),
			expectedPlopDeletions: []string{fixtureconsts.PlopUID1},
		},
		{
			name: "some pods same deployment - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment6, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID3),
			},
			initialPlops: []*storage.ProcessListeningOnPortStorage{
				fixtures.GetPlopStorage1(),
				fixtures.GetPlopStorage2(),
				fixtures.GetPlopStorage3(),
			},
			deployments:           set.NewFrozenStringSet(fixtureconsts.Deployment6),
			pods:                  set.NewFrozenStringSet(fixtureconsts.PodUID3),
			expectedPlopDeletions: []string{fixtureconsts.PlopUID1},
		},
	}
	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			suite.TearDownTest()
			suite.SetupTest()
			// Add deployments if necessary
			deploymentDS, err := deploymentStore.GetTestPostgresDataStore(suite.T(), suite.postgres.DB)
			suite.Nil(err)
			for _, deploymentID := range c.deployments.AsSlice() {
				suite.NoError(deploymentDS.UpsertDeployment(suite.hasAllCtx, &storage.Deployment{Id: deploymentID, ClusterId: fixtureconsts.Cluster1}))
			}

			for _, podID := range c.pods.AsSlice() {
				insertPod := fmt.Sprintf("INSERT INTO pods (id, clusterid) VALUES ('%s', '%s')", podID, fixtureconsts.Cluster1)
				_, err := suite.postgres.DB.Exec(suite.hasWriteCtx, insertPod)
				suite.Nil(err)
			}

			suite.NoError(suite.indicatorDataStore.AddProcessIndicators(suite.hasWriteCtx, c.initialProcesses...))
			countFromDB, err := suite.indicatorDataStore.Count(suite.hasAllCtx, nil)
			suite.NoError(err)
			suite.Equal(len(c.initialProcesses), countFromDB)

			err = suite.store.UpsertMany(suite.hasWriteCtx, c.initialPlops)
			suite.NoError(err)
			plopCount, err := suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
			suite.NoError(err)
			suite.Equal(len(c.initialPlops), plopCount)

			suite.datastore.PruneOrphanedPLOPsByProcessIndicators(suite.hasAllCtx, orphanWindow)

			plopCount, err = suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
			suite.NoError(err)
			suite.Equal(len(c.initialPlops)-len(c.expectedPlopDeletions), plopCount)

		})
	}
}

// RemovePLOPsWithoutProcessIndicatorOrProcessInfo Adds a PLOP with a matching process indicator.
// The process indicator is then deleted. This means that the PLOP has no matching
// process indicator and does not have any process information. A pruning function
// which removes such PLOPs is then called.
func (suite *PLOPDataStoreTestSuite) RemovePLOPsWithoutProcessIndicatorOrProcessInfo() {
	indicators := getIndicators()

	var indicatorIds []string

	for _, indicator := range indicators {
		indicatorIds = append(indicatorIds, indicator.Id)
	}

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	suite.addDeployments()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Cluster1, plopObjects...))

	suite.NoError(suite.indicatorDataStore.RemoveProcessIndicators(
		suite.hasWriteCtx, indicatorIds))

	_, err := suite.datastore.RemovePLOPsWithoutProcessIndicatorOrProcessInfo(suite.hasWriteCtx)
	suite.NoError(err)

	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 0)
}

func (suite *PLOPDataStoreTestSuite) TestRemovePLOPsWithoutPodUID() {
	initialPlops := []*storage.ProcessListeningOnPortStorage{
		fixtures.GetPlopStorage1(),
		fixtures.GetPlopStorage2(),
		fixtures.GetPlopStorage3(),
		fixtures.GetPlopStorage4(),
		fixtures.GetPlopStorage5(),
		fixtures.GetPlopStorage6(),
	}
	expectedPlopDeletions := []string{fixtureconsts.PlopUID1, fixtureconsts.PlopUID2, fixtureconsts.PlopUID3}

	err := suite.store.UpsertMany(suite.hasWriteCtx, initialPlops)
	suite.NoError(err)
	plopCount, err := suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(initialPlops), plopCount)

	prunedCount, err := suite.datastore.RemovePLOPsWithoutPodUID(suite.hasWriteCtx)
	suite.NoError(err)
	suite.Equal(int64(3), prunedCount)

	plopCount, err = suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(initialPlops)-len(expectedPlopDeletions), plopCount)

	ids, err := suite.store.GetIDs(suite.hasReadCtx)
	suite.NoError(err)
	for id := range ids {
		suite.NotContains(expectedPlopDeletions, id)
	}
}

func (suite *PLOPDataStoreTestSuite) addTooMany(plops []*storage.ProcessListeningOnPortFromSensor) {
	batchSize := 30000

	nplops := len(plops)

	for offset := 0; offset < nplops; offset += batchSize {
		end := offset + batchSize
		if end > nplops {
			end = nplops
		}
		err := suite.datastore.AddProcessListeningOnPort(suite.hasWriteCtx, fixtureconsts.Cluster1, plops[offset:end]...)
		suite.NoError(err)
	}
}

func (suite *PLOPDataStoreTestSuite) RemovePLOPsWithoutPodUIDScale(nport int, nprocess int, npod int) {
	plopObjects := suite.makeRandomPlops(nport, nprocess, npod, fixtureconsts.Deployment1)

	plopsWithoutPodUids := 0
	for _, plop := range plopObjects {
		p := rand.Float32()
		if p < 0.5 {
			plop.PodUid = ""
			plopsWithoutPodUids++
		}
	}

	// Add the PLOPs
	suite.addTooMany(plopObjects)

	plopCount, err := suite.store.Count(suite.hasReadCtx, search.EmptyQuery())
	suite.Equal(plopCount, 2*nport*nprocess*npod)
	suite.NoError(err)

	start := time.Now()
	prunedCount, err := suite.datastore.RemovePLOPsWithoutPodUID(suite.hasWriteCtx)
	suite.Equal(int64(plopsWithoutPodUids), prunedCount)
	duration := time.Since(start)
	suite.NoError(err)
	log.Infof("Pruning %d plops took %s", prunedCount, duration)
	_, err = suite.datastore.RemovePLOPsWithoutPodUID(suite.hasWriteCtx)
	suite.NoError(err)
}

func (suite *PLOPDataStoreTestSuite) TestRemovePLOPsWithoutPodUIDScale27K() {
	nport := 30
	nprocess := 30
	npod := 30

	suite.RemovePLOPsWithoutPodUIDScale(nport, nprocess, npod)
}

func (suite *PLOPDataStoreTestSuite) TestRemovePLOPsWithoutPodUIDScale125K() {
	nport := 50
	nprocess := 50
	npod := 50

	suite.RemovePLOPsWithoutPodUIDScale(nport, nprocess, npod)
}

func (suite *PLOPDataStoreTestSuite) TestRemovePLOPsWithoutPodUIDScaleRaceCondition() {
	var wg sync.WaitGroup
	wg.Add(1)

	running := true
	plopsWithoutPodUids := 0
	go func() {
		defer wg.Done()
		iterations := 3
		for i := 0; i < iterations; i++ {
			nport := 30
			nprocess := 30
			npod := 30
			plopObjects := suite.makeRandomPlops(nport, nprocess, npod, fixtureconsts.Deployment1)

			for _, plop := range plopObjects {
				p := rand.Float32()
				if p < 0.5 {
					plop.PodUid = ""
					plopsWithoutPodUids += 1
				} else {
					plop.PodUid = uuid.NewV4().String()
				}
			}

			// Add the open PLOPs
			suite.addTooMany(plopObjects)

			// Close the PLOPs
			// This is so that UpsertMany will trigger deletes
			for _, plop := range plopObjects {
				plop.CloseTimestamp = protoconv.ConvertTimeToTimestamp(time.Now())
			}

			// Add the closed PLOPs
			suite.addTooMany(plopObjects)
		}
	}()

	totalPrunedCount := 0

	var wgPrune sync.WaitGroup
	wgPrune.Add(1)
	go func() {
		defer wgPrune.Done()
		for running {
			prunedCount, err := suite.datastore.RemovePLOPsWithoutPodUID(suite.hasWriteCtx)
			suite.NoError(err)
			totalPrunedCount += int(prunedCount)
		}
	}()

	wg.Wait()

	running = false

	wgPrune.Wait()

	prunedCount, err := suite.datastore.RemovePLOPsWithoutPodUID(suite.hasWriteCtx)
	suite.NoError(err)
	totalPrunedCount += int(prunedCount)

	// Each PLOP is opened and closed. If a PLOP is opened then pruned, then closed,
	// and pruned again, then it will be pruned twice. If a PLOP is opened, then closed,
	// and pruned, then it will be pruned once. Therefore the number of rows pruned will
	// be between the number of PLOPs that don't have poduids that were added and twice
	// that number.
	suite.GreaterOrEqual(int(plopsWithoutPodUids), totalPrunedCount/2)
	suite.LessOrEqual(int(plopsWithoutPodUids), totalPrunedCount)
}
