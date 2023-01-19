package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
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

	mockCtrl *gomock.Controller
}

func (suite *PLOPDataStoreTestSuite) SetupSuite() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		suite.T().Skip("Skip PLOP tests if postgres is disabled")
		suite.T().SkipNow()
	}
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
	suite.store = postgresStore.NewFullStore(suite.postgres.Pool)

	indicatorStorage := processIndicatorStorage.New(suite.postgres.Pool)
	indicatorIndexer := processIndicatorStorage.NewIndexer(suite.postgres.Pool)
	indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

	suite.indicatorDataStore, _ = processIndicatorDataStore.New(
		indicatorStorage, suite.store, indicatorIndexer, indicatorSearcher, nil)
	processIndicatorDataStore.Singleton()
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.datastore = New(suite.store, suite.indicatorDataStore)
}

func (suite *PLOPDataStoreTestSuite) TearDownTest() {
	suite.postgres.Teardown(suite.T())
	suite.mockCtrl.Finish()
}

// TestPLOPAdd: Happy path for ProcessListeningOnPort, one PLOP object is added
// with a correct process indicator reference and could be fetched later.
func (suite *PLOPDataStoreTestSuite) TestPLOPAdd() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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
		PodId:         fixtureconsts.PodUID1,
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
	processInfo := []*storage.ProcessIndicatorUniqueKey{}
	err = suite.datastore.WalkAll(suite.hasWriteCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			if plop.GetProcess() != nil {
				processInfo = append(processInfo, plop.GetProcess())
			}
			return nil
		})
	suite.NoError(err)
	suite.Len(processInfo, 0)
}

// TestPLOPAddClosed: Happy path for ProcessListeningOnPort closing, one PLOP object is added
// with a correct process indicator reference and CloseTimestamp set. It will
// be exluded from the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosed() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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

	// It's being closed and excluded from the API response
	suite.Len(newPlops, 0)
}

// TestPLOPReopen: One PLOP object is added with a correct process indicator
// reference and CloseTimestamp set to nil. It will reopen an existing PLOP and
// present in the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPReopen() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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
		PodId:         fixtureconsts.PodUID1,
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
	plopsFromDB := []*storage.ProcessListeningOnPortStorage{}
	err = suite.datastore.WalkAll(suite.hasWriteCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			plopsFromDB = append(plopsFromDB, plop)
			return nil
		})
	suite.NoError(err)
	suite.Len(plopsFromDB, 1)
}

// TestPLOPCloseSameTimestamp: One PLOP object is added with a correct process
// indicator reference and CloseTimestamp set to the same as existing one.
func (suite *PLOPDataStoreTestSuite) TestPLOPCloseSameTimestamp() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjectsActive := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

	plopObjectsClosed := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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
		suite.hasWriteCtx, plopObjectsClosed...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	// It's being closed and excluded from the API response
	suite.Len(newPlops, 0)

	// Verify that PLOP object was updated and no new records were created
	plopsFromDB := []*storage.ProcessListeningOnPortStorage{}
	err = suite.datastore.WalkAll(suite.hasWriteCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			plopsFromDB = append(plopsFromDB, plop)
			return nil
		})
	suite.NoError(err)
	suite.Len(plopsFromDB, 1)
}

// TestPLOPAddClosedSameBatch: One PLOP object is added with a correct process
// indicator reference with and without CloseTimestamp set in the same batch.
// This will excercise logic of batch normalization. It will be exluded from
// the API result.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedSameBatch() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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

	// It's being closed and excluded from the API response
	suite.Len(newPlops, 0)
}

// TestPLOPAddClosedWithoutActive: one PLOP object is added with a correct
// process indicator reference and CloseTimestamp set, without having
// previously active PLOP. Will be stored in the db as closed and excluded from
// the API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedWithoutActive() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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

	suite.Len(newPlops, 0)
}

// TestPLOPAddNoIndicator: A PLOP object with a wrong process indicator
// reference. It's being stored in the database, but without the reference will
// not be fetched via API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddNoIndicator() {
	plopObjects := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)
}

// TestPLOPAddClosedNoIndicator: A PLOP object with a wrong process indicator
// reference and CloseTimestamp set. It's being stored in the database, but
// without the reference will not be fetched via API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddClosedNoIndicator() {
	plopObjects := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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
	processInfo := []*storage.ProcessIndicatorUniqueKey{}
	err = suite.datastore.WalkAll(suite.hasWriteCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			if plop.GetProcess() != nil {
				processInfo = append(processInfo, plop.GetProcess())
			}
			return nil
		})
	suite.NoError(err)
	suite.Len(processInfo, 1)
	suite.Equal(processInfo[0], plopObjects[0].Process)
}

// TestPLOPAddMultipleIndicators: A PLOP object is added with a valid reference
// that somehow matches two process indicator records. Such object could be
// fetched from the API with only one process indicator attached (one is going
// to be ignored).
func (suite *PLOPDataStoreTestSuite) TestPLOPAddMultipleIndicators() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{
		{
			Port:           1234,
			Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
			CloseTimestamp: nil,
			Process: &storage.ProcessIndicatorUniqueKey{
				PodId:               fixtureconsts.PodUID1,
				ContainerName:       "test_container1",
				ProcessName:         "test_process1",
				ProcessArgs:         "test_arguments1",
				ProcessExecFilePath: "test_path1",
			},
		},
	}

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
		PodId:         fixtureconsts.PodUID1,
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
}

func (suite *PLOPDataStoreTestSuite) TestPLOPAddOpenThenCloseAndOpenSameBatch() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	openPlopObject := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

	closedPlopObject := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

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

	// It's being closed and excluded from the API response
	suite.Len(newPlops, 1)
}

func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseThenCloseAndOpenSameBatch() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	openPlopObject := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

	closedPlopObject := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

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

	// It's being closed and excluded from the API response
	suite.Len(newPlops, 0)
}

func (suite *PLOPDataStoreTestSuite) TestPLOPAddCloseBatchOutOfOrder() {
	testNamespace := "test_namespace"

	indicators := []*storage.ProcessIndicator{
		{
			Id:            fixtureconsts.ProcessIndicatorID1,
			DeploymentId:  fixtureconsts.Deployment1,
			PodId:         fixtureconsts.PodUID1,
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
			PodId:         fixtureconsts.PodUID2,
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

	openPlopObject := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: nil,
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

	time1 := time.Now()
	time2 := time.Now().Local().Add(time.Hour * time.Duration(1))
	time3 := time.Now().Local().Add(time.Hour * time.Duration(2))

	closedPlopObject1 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time1),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

	closedPlopObject2 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time2),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

	closedPlopObject3 := storage.ProcessListeningOnPortFromSensor{
		Port:           1234,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
		CloseTimestamp: protoconv.ConvertTimeToTimestamp(time3),
		Process: &storage.ProcessIndicatorUniqueKey{
			PodId:               fixtureconsts.PodUID1,
			ContainerName:       "test_container1",
			ProcessName:         "test_process1",
			ProcessArgs:         "test_arguments1",
			ProcessExecFilePath: "test_path1",
		},
	}

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{&closedPlopObject1}

	batchPlopObjects := []*storage.ProcessListeningOnPortFromSensor{
		&closedPlopObject3,
		&openPlopObject,
		&closedPlopObject2,
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

	// It's being closed and excluded from the API response
	suite.Len(newPlops, 0)

	plopsFromDB := []*storage.ProcessListeningOnPortStorage{}
	err = suite.datastore.WalkAll(suite.hasWriteCtx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			plopsFromDB = append(plopsFromDB, plop)
			return nil
		})
	suite.NoError(err)
	suite.Len(plopsFromDB, 1)
	suite.Equal(plopsFromDB[0].CloseTimestamp, closedPlopObject3.CloseTimestamp)
}
