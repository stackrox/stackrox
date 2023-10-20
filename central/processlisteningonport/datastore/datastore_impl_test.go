//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
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
}

func (suite *PLOPDataStoreTestSuite) SetupTest() {
	suite.postgres = pgtest.ForT(suite.T())
	suite.store = postgresStore.NewFullStore(suite.postgres.DB)

	indicatorStorage := processIndicatorStorage.New(suite.postgres.DB)
	indicatorIndexer := processIndicatorStorage.NewIndexer(suite.postgres.DB)
	indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

	suite.indicatorDataStore, _ = processIndicatorDataStore.New(
		indicatorStorage, suite.store, indicatorSearcher, nil)
	suite.datastore = New(suite.store, suite.indicatorDataStore)
}

func (suite *PLOPDataStoreTestSuite) TearDownTest() {
	suite.postgres.Teardown(suite.T())
}

func (suite *PLOPDataStoreTestSuite) getPlopsFromDB() []*storage.ProcessListeningOnPortStorage {
	plopsFromDB := []*storage.ProcessListeningOnPortStorage{}
	err := suite.datastore.WalkAll(suite.hasWriteCtx,
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
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodName1,
			PodUid:        fixtureconsts.PodUID1,
			ClusterId:     fixtureconsts.Cluster1,
			ContainerName: "test_container1",
			Namespace:     testNamespace,

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
			Namespace:     testNamespace,

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
	}
)

// TestPLOPAdd: Happy path for ProcessListeningOnPort, one PLOP object is added
// with a correct process indicator reference and could be fetched later.
func (suite *PLOPDataStoreTestSuite) TestPLOPAdd() {
	testNamespace := "test_namespace"

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
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
		DeploymentId:       plopObjects[0].GetDeploymentId(),
		PodUid:             plopObjects[0].GetPodUid(),
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosed: Happy path for ProcessListeningOnPort closing, one PLOP object is added
// with a correct process indicator reference and CloseTimestamp set. It will
// be exluded from the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosed() {

	indicators := getIndicators()

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsActive...))

	// Close PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsClosed...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenTwice: Add the same open PLOP twice
// There should only be one PLOP in the storage and one
// PLOP returned by the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenTwice() {
	testNamespace := "test_namespace"

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Add the same PLOP again
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddCloseTwice: Add the same closed PLOP twice
// There should only be one PLOP in the storage
func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseTwice() {
	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Add the same PLOP again
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPReopen: One PLOP object is added with a correct process indicator
// reference and CloseTimestamp set to nil. It will reopen an existing PLOP and
// present in the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPReopen() {
	testNamespace := "test_namespace"

	indicators := getIndicators()

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsActive...))

	// Close PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsClosed...))

	// Reopen PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsActive...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// The PLOP is reported since it is in the open state
	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPCloseSameTimestamp: One PLOP object is added with a correct process
// indicator reference and CloseTimestamp set to the same as existing one.
func (suite *PLOPDataStoreTestSuite) TestPLOPCloseSameTimestamp() {

	indicators := getIndicators()

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsActive...))

	// Close PLOP objects
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsClosed...))

	// Send same close event again
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjectsClosed...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosedSameBatch: One PLOP object is added with a correct process
// indicator reference with and without CloseTimestamp set in the same batch.
// This will excercise logic of batch normalization. It will be exluded from
// the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedSameBatch() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject, &closedPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosedWithoutActive: one PLOP object is added with a correct
// process indicator reference and CloseTimestamp set, without having
// previously active PLOP. Will be stored in the db as closed and excluded from
// the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedWithoutActive() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Confirm that the database is empty before anything is inserted into it
	plopsFromDB := suite.getPlopsFromDB()
	suite.Len(plopsFromDB, 0)

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddNoIndicator: A PLOP object with a wrong process indicator
// reference. It's being stored in the database and will be returned by
// the API even though it is not matched to a process.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddNoIndicator() {

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

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
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddClosedNoIndicator: A PLOP object with a wrong process indicator
// reference and CloseTimestamp set. It's stored in the database, but
// as it is closed it will not be returned by the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedNoIndicator() {

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenNoIndicatorThenClose Adds an open PLOP object with no matching
// indicator. Adds an indicator and then closes the PLOP
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenNoIndicatorThenClose() {

	openPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, openPlopObjects...))

	indicators := getIndicators()

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Add closed PLOP now with a matching indicator
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, closedPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenAndClosedNoIndicator: An open PLOP object with no matching
// process indicator is sent. Then the PLOP object is closed. We expect the
// database to contain one PLOP and the database to return nothing, because
// it is closed.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenAndClosedNoIndicator() {
	openPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}
	closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Add open PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, openPlopObjects...))

	// Add closed PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, closedPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddMultipleIndicators: A PLOP object is added with a valid reference
// that matches one of two process indicators
func (suite *PLOPDataStoreTestSuite) TestPLOPAddMultipleIndicators() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodName1,
			PodUid:        fixtureconsts.PodUID1,
			ClusterId:     fixtureconsts.Cluster1,
			ContainerName: "test_container1",
			Namespace:     testNamespace,

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
			Namespace:     testNamespace,

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

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPAddOpenThenCloseAndOpenSameBatch Sends an open PLOP witha matching indicator.
// Then closes and opens the PLOP object in the same batch.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenThenCloseAndOpenSameBatch() {

	indicators := getIndicators()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&closedPlopObject,
		&openPlopObject,
	}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
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

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
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
		&closedPlopObject3,
		&openPlopObject,
		&closedPlopObject2,
		&openPlopObject,
	}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Add the same PLOP in an open and closed state multiple times
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
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

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, indicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Add the same PLOP in an open and closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, batchPlopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
}

// TestPLOPDeleteAndCreateDeployment: Creates a deployment with PLOP and
// matching process indicator. Closes the PLOP and deletes the process indicator.
// Creates the PLOP with a new DeploymentId and matching process indicator.
func (suite *PLOPDataStoreTestSuite) TestPLOPDeleteAndCreateDeployment() {
	testNamespace := "test_namespace"

	initialIndicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodName1,
			PodUid:        fixtureconsts.PodUID1,
			ClusterId:     fixtureconsts.Cluster1,
			ContainerName: "test_container1",
			Namespace:     testNamespace,

			Signal: &storage.ProcessSignal{
				Name:         "test_process1",
				Args:         "test_arguments1",
				ExecFilePath: "test_path1",
			},
		},
	}

	id.SetIndicatorID(initialIndicators[0])

	openPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}

	// Prepare indicators for FK
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, initialIndicators...))

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, openPlopObjects...))

	closedPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject}

	// Add the same PLOP in a closed state
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, closedPlopObjects...))

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
	}

	suite.Equal(expectedPlopStorage1, plopsFromDB1[0])

	// Delete the indicator
	suite.NoError(suite.indicatorDataStore.RemoveProcessIndicators(
		suite.hasWriteCtx, idsToDelete))

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
			Namespace:     testNamespace,

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
	}

	newOpenPlopObjects := []*storage.ProcessListeningOnPortFromSensor{&newOpenPlopObject}

	// Add new indicator
	suite.NoError(suite.indicatorDataStore.AddProcessIndicators(
		suite.hasWriteCtx, newIndicators...))

	// Add the PLOP with the new DeploymentId
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, newOpenPlopObjects...))

	// Fetch inserted PLOP back from the new deployment
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment2)
	suite.NoError(err)

	// It's open and included in the API response for the new deployment
	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment2,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
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
		suite.hasWriteCtx, fixtureconsts.Deployment1)
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
	}

	suite.Equal(expectedPlopStorage, newPlopsFromDB[0])
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
	}

	// It is not possible to add a PLOP from sensor with no process info
	// so upsert directly to the database. In the tests when a process indicator
	// is deleted the PLOP is also deleted from the database. This does not seem
	// to be the case in reality.
	suite.NoError(suite.store.Upsert(
		suite.hasWriteCtx, plopStorage))

	// Fetch inserted PLOP back
	// It doesn't appear in the API, because it has no process info
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)

	// Verify that the PLOP is in the database
	newPlopsFromDB := suite.getPlopsFromDB()
	suite.Len(newPlopsFromDB, 1)

	suite.Equal(plopStorage, newPlopsFromDB[0])
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
	}

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&plop1, &plop2}

	// Add PLOP referencing those indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

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
		},
	}

	expectedPlopStorageMap := getPlopMap(expectedPlopStorage)

	for key, expectedPlop := range expectedPlopStorageMap {
		// We cannot know the Id in advance so set it here.
		expectedPlop.Id = plopMap[key].Id
		suite.Equal(expectedPlop, plopMap[key])
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
	}

	suite.Equal(expectedPlopStorage1, newPlopsFromDB2[0])

}

func makeRandomString(length int) string {
        var charset = []byte("asdfqwert")
        randomString := make([]byte, length)
        for i := range randomString {
                randomString[i] = charset[rand.Intn(len(charset))]
        }
        return string(randomString)
}

func (suite *PLOPDataStoreTestSuite) makeRandomPlops(nport int, nprocess int, npod int, deployment string) {
        count := 0

	batchSize := 100

	nplops := 2*nprocess*npod*nport

	if batchSize > nplops {
		batchSize = nplops
	}

        plops := make([]*storage.ProcessListeningOnPortFromSensor, batchSize)
        for podIdx := 0; podIdx < npod; podIdx++ {
                podID := makeRandomString(10)
                podUID := makeRandomString(10)
                for processIdx := 0; processIdx < nprocess; processIdx++ {
                        execFilePath := makeRandomString(10)
                        for port := 0; port < nport; port++ {

                                plopTCP := &storage.ProcessListeningOnPortFromSensor{
                                        Port:     uint32(port),
                                        Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					CloseTimestamp: nil,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               podID,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: execFilePath,
					},
					DeploymentId: deployment,
					PodUid:       podUID,
                                }
                                plopUDP := &storage.ProcessListeningOnPortFromSensor{
                                        Port:     uint32(port),
                                        Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
					CloseTimestamp: nil,
					Process: &storage.ProcessIndicatorUniqueKey{
						PodId:               podID,
						ContainerName:       "test_container1",
						ProcessName:         "test_process1",
						ProcessArgs:         "test_arguments1",
						ProcessExecFilePath: execFilePath,
					},
					DeploymentId: deployment,
					PodUid:       podUID,
                                }
                                plops[count] = plopTCP
                                count++
                                plops[count] = plopUDP
                                count++
				if count == batchSize {
					suite.NoError(suite.datastore.AddProcessListeningOnPort(
						suite.hasWriteCtx, plops...))
					count = 0
				}
                        }
                }
        }
}

func (suite *PLOPDataStoreTestSuite) TestSort1000000() {
        nport := 100
        nprocess := 100
        npod := 100

        suite.makeRandomPlops(nport, nprocess, npod, fixtureconsts.Deployment1)

        startTime := time.Now()
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)
        duration := time.Since(startTime)

	fmt.Printf("Fetching %d plops %s took\n", len(newPlops), duration)

}

