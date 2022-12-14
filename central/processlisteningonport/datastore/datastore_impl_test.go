package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
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

func (suite *PLOPDataStoreTestSuite) SetupTest() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
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

	suite.postgres = pgtest.ForT(suite.T())
	suite.store = postgresStore.New(suite.postgres.Pool)

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
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}

	suite.postgres.Teardown(suite.T())
	suite.mockCtrl.Finish()
}

// TestPLOPAdd: Happy path for ProcessListeningOnPort, one PLOP object is added
// with a correct process indicator reference and could be fetched later.
func (suite *PLOPDataStoreTestSuite) TestPLOPAdd() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}

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

	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
		Endpoints: []*storage.ProcessListeningOnPort_Endpoint{
			&storage.ProcessListeningOnPort_Endpoint{
				Port:     1234,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
		},
		Signal: &storage.ProcessSignal{
			Name:         "test_process1",
			Args:         "test_arguments1",
			ExecFilePath: "test_path1",
		},
	})
}

// TestPLOPAddNoIndicator: A PLOP object with a wrong process indicator
// reference. It's being stored in the database, but without the reference will
// not be fetched via API.
func (suite *PLOPDataStoreTestSuite) TestPLOPAddNoIndicator() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
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

	// Add PLOP referencing non existing indicators
	suite.NoError(suite.datastore.AddProcessListeningOnPort(
		suite.hasWriteCtx, plopObjects...))

	// Fetch inserted PLOP back
	newPlops, err := suite.datastore.GetProcessListeningOnPort(
		suite.hasWriteCtx, fixtureconsts.Deployment1)
	suite.NoError(err)

	suite.Len(newPlops, 0)
}

// TestPLOPAddMultipleIndicators: A PLOP object is added with a valid reference
// that somehow matches two process indicator records. Such object could be
// fetched from the API with only one process indicator attached (one is going
// to be ignored).
func (suite *PLOPDataStoreTestSuite) TestPLOPAddMultipleIndicators() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return
	}

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

	suite.Len(newPlops, 1)
	suite.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodUID1,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     testNamespace,
		Endpoints: []*storage.ProcessListeningOnPort_Endpoint{
			&storage.ProcessListeningOnPort_Endpoint{
				Port:     1234,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
		},
		Signal: &storage.ProcessSignal{
			Name:         "test_process1",
			Args:         "test_arguments1",
			ExecFilePath: "test_path1",
		},
	})
}
