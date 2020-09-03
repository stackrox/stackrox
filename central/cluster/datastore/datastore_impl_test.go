package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterIndexMocks "github.com/stackrox/rox/central/cluster/index/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/store/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	netEntityMocks "github.com/stackrox/rox/central/networkflow/datastore/entities/mocks"
	netFlowsMocks "github.com/stackrox/rox/central/networkflow/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/processor/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	graphMocks "github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const fakeClusterID = "FAKECLUSTERID"

func TestClusterDataStore(t *testing.T) {
	suite.Run(t, new(ClusterDataStoreTestSuite))
}

type ClusterDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	clusters            *clusterMocks.MockClusterStore
	healthStatuses      *clusterMocks.MockClusterHealthStore
	indexer             *clusterIndexMocks.MockIndexer
	clusterDataStore    DataStore
	namespaceDataStore  *namespaceMocks.MockDataStore
	deploymentDataStore *deploymentMocks.MockDataStore
	nodeDataStore       *nodeMocks.MockGlobalDataStore
	secretDataStore     *secretMocks.MockDataStore
	flowsDataStore      *netFlowsMocks.MockClusterDataStore
	netEntityDataStore  *netEntityMocks.MockEntityDataStore
	connMgr             *connectionMocks.MockManager
	alertDataStore      *alertMocks.MockDataStore
	riskDataStore       *riskMocks.MockDataStore
	mockCtrl            *gomock.Controller
	notifierMock        *notifierMocks.MockProcessor
	mockProvider        *graphMocks.MockProvider
}

func (suite *ClusterDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.clusters = clusterMocks.NewMockClusterStore(suite.mockCtrl)
	suite.healthStatuses = clusterMocks.NewMockClusterHealthStore(suite.mockCtrl)
	suite.indexer = clusterIndexMocks.NewMockIndexer(suite.mockCtrl)

	suite.namespaceDataStore = namespaceMocks.NewMockDataStore(suite.mockCtrl)
	suite.deploymentDataStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.alertDataStore = alertMocks.NewMockDataStore(suite.mockCtrl)
	suite.nodeDataStore = nodeMocks.NewMockGlobalDataStore(suite.mockCtrl)
	suite.secretDataStore = secretMocks.NewMockDataStore(suite.mockCtrl)
	suite.flowsDataStore = netFlowsMocks.NewMockClusterDataStore(suite.mockCtrl)
	suite.netEntityDataStore = netEntityMocks.NewMockEntityDataStore(suite.mockCtrl)
	suite.riskDataStore = riskMocks.NewMockDataStore(suite.mockCtrl)
	suite.connMgr = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.notifierMock = notifierMocks.NewMockProcessor(suite.mockCtrl)
	suite.mockProvider = graphMocks.NewMockProvider(suite.mockCtrl)

	suite.nodeDataStore.EXPECT().GetAllClusterNodeStores(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	suite.clusters.EXPECT().Walk(gomock.Any()).Return(nil)
	suite.healthStatuses.EXPECT().WalkAllWithID(gomock.Any()).Return(nil)
	suite.indexer.EXPECT().AddClusters(nil).Return(nil)

	var err error
	suite.clusterDataStore, err = New(
		suite.clusters,
		suite.healthStatuses,
		suite.indexer,
		suite.alertDataStore,
		suite.namespaceDataStore,
		suite.deploymentDataStore,
		suite.nodeDataStore,
		suite.secretDataStore,
		suite.flowsDataStore,
		suite.netEntityDataStore,
		suite.connMgr,
		suite.notifierMock,
		suite.mockProvider,
		ranking.NewRanker(),
	)
	suite.NoError(err)
}

func (suite *ClusterDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

//// Test that when the cluster we try to remove does not exist, we return an error.
func (suite *ClusterDataStoreTestSuite) TestHandlesClusterDoesNotExist() {
	// Return false for the cluster not existing.
	suite.clusters.EXPECT().Get(fakeClusterID).Return((*storage.Cluster)(nil), false, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID, nil)
	suite.Error(err, "expected an error since the cluster did not exist")
}

// Test that when we cannot fetch a cluster, we return the error from the DB.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingCluster() {
	// Return an error trying to fetch the cluster.
	expectedErr := errors.New("issues need tissues")
	suite.clusters.EXPECT().Get(fakeClusterID).Return((*storage.Cluster)(nil), true, expectedErr)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID, nil)
	suite.Equal(expectedErr, err)
}

func (suite *ClusterDataStoreTestSuite) TestRemoveCluster() {
	testCluster := &storage.Cluster{Id: fakeClusterID}
	testDeployments := []search.Result{{ID: "fakeDeployment"}}
	testAlerts := []*storage.Alert{{}}
	testSecrets := []*storage.ListSecret{{}}
	suite.clusters.EXPECT().Get(fakeClusterID).Return(testCluster, true, nil)
	suite.clusters.EXPECT().Delete(fakeClusterID).Return(nil)
	suite.indexer.EXPECT().DeleteCluster(fakeClusterID).Return(nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
	suite.namespaceDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil)
	suite.deploymentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(testDeployments, nil)
	suite.alertDataStore.EXPECT().SearchRawAlerts(gomock.Any(), gomock.Any()).Return(testAlerts, nil)
	suite.alertDataStore.EXPECT().MarkAlertStale(gomock.Any(), gomock.Any()).Return(nil)
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), gomock.Any()).Return()
	suite.deploymentDataStore.EXPECT().RemoveDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	suite.nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	suite.secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(testSecrets, nil)
	if features.NetworkGraphExternalSrcs.Enabled() {
		suite.netEntityDataStore.EXPECT().DeleteExternalNetworkEntitiesForCluster(gomock.Any(), fakeClusterID).Return(nil)
	}
	suite.secretDataStore.EXPECT().RemoveSecret(gomock.Any(), gomock.Any()).Return(nil)

	done := concurrency.NewSignal()
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID, &done)
	suite.NoError(err)
	suite.True(concurrency.WaitWithTimeout(&done, 10*time.Second))
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesGet() {
	suite.clusters.EXPECT().Get(gomock.Any()).Times(0)

	cluster, exists, err := suite.clusterDataStore.GetCluster(suite.hasNoneCtx, "hkjddjhk")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists)
	suite.Nil(cluster, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsGet() {
	suite.clusters.EXPECT().Get(gomock.Any()).Return(nil, false, nil)

	_, _, err := suite.clusterDataStore.GetCluster(suite.hasReadCtx, "An Id")
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().Get(gomock.Any()).Return(nil, false, nil)

	_, _, err = suite.clusterDataStore.GetCluster(suite.hasWriteCtx, "beef")
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesGetAll() {
	suite.clusters.EXPECT().GetMany([]string{}).Return(nil, nil, nil)
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any()).Return([]*storage.ClusterHealthStatus{}, []int{}, nil)

	clusters, err := suite.clusterDataStore.GetClusters(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Empty(clusters, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsGetAll() {
	suite.clusters.EXPECT().Walk(gomock.Any()).Return(nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any()).Return([]*storage.ClusterHealthStatus{}, []int{}, nil)

	_, err := suite.clusterDataStore.GetClusters(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().Walk(gomock.Any()).Return(nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any()).Return([]*storage.ClusterHealthStatus{}, []int{}, nil)

	_, err = suite.clusterDataStore.GetClusters(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesCount() {
	suite.clusters.EXPECT().Count().Times(0)
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	count, err := suite.clusterDataStore.CountClusters(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return 0 without access")
	suite.Zero(count, "expected return value to be 0")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsCount() {
	suite.clusters.EXPECT().Count().Return(99, nil)

	_, err := suite.clusterDataStore.CountClusters(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().Count().Return(42, nil)

	_, err = suite.clusterDataStore.CountClusters(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesAdd() {
	suite.clusters.EXPECT().Upsert(gomock.Any()).Times(0)

	_, err := suite.clusterDataStore.AddCluster(suite.hasNoneCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")

	_, err = suite.clusterDataStore.AddCluster(suite.hasReadCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsAdd() {
	suite.clusters.EXPECT().Upsert(gomock.Any()).Return(nil)
	suite.indexer.EXPECT().AddCluster(gomock.Any()).Return(nil)
	suite.flowsDataStore.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(suite.mockCtrl), nil)

	_, err := suite.clusterDataStore.AddCluster(suite.hasWriteCtx, &storage.Cluster{Name: "blah"})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesUpdate() {
	suite.clusters.EXPECT().Upsert(gomock.Any()).Times(0)

	err := suite.clusterDataStore.UpdateCluster(suite.hasNoneCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.UpdateCluster(suite.hasReadCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsUpdate() {
	suite.clusters.EXPECT().Get(gomock.Any()).Return(&storage.Cluster{Id: "1", Name: "blah"}, true, nil)
	suite.clusters.EXPECT().Upsert(gomock.Any()).Return(nil)
	suite.indexer.EXPECT().AddCluster(gomock.Any()).Return(nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)

	err := suite.clusterDataStore.UpdateCluster(suite.hasWriteCtx, &storage.Cluster{Id: "1", Name: "blah"})
	suite.NoError(err, "expected no error trying to write with permissions")

	suite.clusters.EXPECT().Get(gomock.Any()).Return(&storage.Cluster{Id: "1", Name: "blah"}, true, nil)

	err = suite.clusterDataStore.UpdateCluster(suite.hasWriteCtx, &storage.Cluster{Id: "1", Name: "blahDiff"})
	suite.Error(err, "expected error trying to rename cluster")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesRemove() {
	suite.clusters.EXPECT().Delete(gomock.Any()).Times(0)

	err := suite.clusterDataStore.RemoveCluster(suite.hasNoneCtx, "jiogserlksd", nil)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.RemoveCluster(suite.hasReadCtx, "vflkjdf", nil)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsRemove() {
	// This is a weird thing for store.Get() to return but we're only testing auth here
	suite.clusters.EXPECT().Get("poiuytre").Return(nil, true, nil)
	suite.clusters.EXPECT().Delete(gomock.Any()).Return(nil)
	suite.indexer.EXPECT().DeleteCluster(gomock.Any()).Return(nil)
	suite.namespaceDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.deploymentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	if features.NetworkGraphExternalSrcs.Enabled() {
		suite.netEntityDataStore.EXPECT().DeleteExternalNetworkEntitiesForCluster(gomock.Any(), gomock.Any()).Return(nil)
	}
	suite.secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)

	done := concurrency.NewSignal()
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, "poiuytre", &done)
	suite.NoError(err, "expected no error trying to write with permissions")
	suite.True(concurrency.WaitWithTimeout(&done, 10*time.Second))
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesUpdateClusterStatus() {
	err := suite.clusterDataStore.UpdateClusterStatus(suite.hasNoneCtx, "F", &storage.ClusterStatus{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.UpdateClusterStatus(suite.hasReadCtx, "IDK", &storage.ClusterStatus{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsUpdateClusterStatus() {
	suite.clusters.EXPECT().Get(gomock.Any()).Return(&storage.Cluster{Id: "qwerty", Name: "blah"}, true, nil)
	suite.clusters.EXPECT().Upsert(&storage.Cluster{Id: "qwerty", Name: "blah", Status: &storage.ClusterStatus{}}).Return(nil)

	err := suite.clusterDataStore.UpdateClusterStatus(suite.hasWriteCtx, "qwerty", &storage.ClusterStatus{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesSearch() {
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	clusters, err := suite.clusterDataStore.Search(suite.hasNoneCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Nil(clusters, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsSearch() {
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	_, err := suite.clusterDataStore.Search(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	_, err = suite.clusterDataStore.Search(suite.hasWriteCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestPopulateClusterHealthInfo() {
	t := time.Now()
	ts := protoconv.ConvertTimeToTimestamp(t)
	ids := []string{"1", "2", "3", "4", "5", "6"}
	existingHealths := []*storage.ClusterHealthStatus{
		{
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			LastContact:        ts,
		},
	}
	results := []search.Result{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"}, {ID: "6"}}
	clusters := []*storage.Cluster{{Id: "1"}, {Id: "2"}, {Id: "3"}, {Id: "4"}, {Id: "5"}, {Id: "6"}}
	expected := []*storage.Cluster{
		{
			Id:       "1",
			Priority: 1,
		},
		{
			Id: "2",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id:       "3",
			Priority: 1,
		},
		{
			Id: "4",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id: "5",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id:       "6",
			Priority: 1,
		},
	}

	suite.indexer.EXPECT().Search(gomock.Any()).Return(results, nil)
	suite.clusters.EXPECT().GetMany(ids).Return(clusters, []int{}, nil)
	suite.healthStatuses.EXPECT().GetMany(ids).Return(existingHealths, []int{0, 2, 5}, nil)

	actuals, err := suite.clusterDataStore.SearchRawClusters(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Equal(expected, actuals)

	// none are missing
	existingHealths = []*storage.ClusterHealthStatus{
		{
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
			LastContact:        ts,
		},
		{
			SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
			LastContact:        ts,
		},
	}
	expected = []*storage.Cluster{
		{
			Id: "1",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id: "2",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id: "3",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id: "4",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id: "5",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_DEGRADED,
				LastContact:        ts,
			},
			Priority: 1,
		},
		{
			Id: "6",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        ts,
			},
			Priority: 1,
		},
	}

	suite.indexer.EXPECT().Search(gomock.Any()).Return(results, nil)
	suite.clusters.EXPECT().GetMany(ids).Return(clusters, []int{}, nil)
	suite.healthStatuses.EXPECT().GetMany(ids).Return(existingHealths, []int{}, nil)

	actuals, err = suite.clusterDataStore.SearchRawClusters(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Equal(expected, actuals)
}

func (suite *ClusterDataStoreTestSuite) TestUpdateClusterHealth() {
	t1 := time.Now()
	t3 := time.Now().Add(-30 * time.Minute)
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts3 := protoconv.ConvertTimeToTimestamp(t3)

	cases := []struct {
		name      string
		oldHealth *storage.ClusterHealthStatus
		newHealth *storage.ClusterHealthStatus
		cluster   *storage.Cluster
		skipIndex bool
	}{
		{
			name: "status change: first check-in, must index",
			oldHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNINITIALIZED,
			},
			newHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts1,
			},
			cluster: &storage.Cluster{
				Id: "1",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
					LastContact:        ts1,
				},
			},
			skipIndex: false,
		},
		{
			name: "status change: unhealthy to healthy, must index",
			oldHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        ts3,
			},
			newHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts1,
			},
			cluster: &storage.Cluster{
				Id: "2",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
					LastContact:        ts1,
				},
			},
			skipIndex: false,
		},
		{
			name: "no status change: healthy, skip index",
			oldHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts1,
			},
			newHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts1,
			},
			cluster: &storage.Cluster{
				Id: "3",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
					LastContact:        ts1,
				},
			},
			skipIndex: true,
		},
		{
			name: "no status change: unhealthy, skip index",
			oldHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				LastContact:           ts3,
			},
			newHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				OverallHealthStatus:   storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:           ts3,
			},
			cluster: &storage.Cluster{
				Id: "4",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
					CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
					OverallHealthStatus:   storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:           ts3,
				},
			},
			skipIndex: true,
		},
		{
			name: "status change: degraded to unhealthy, must index",
			oldHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_DEGRADED,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				LastContact:           ts3,
			},
			newHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				OverallHealthStatus:   storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:           ts3,
			},
			cluster: &storage.Cluster{
				Id: "5",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
					CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
					OverallHealthStatus:   storage.ClusterHealthStatus_UNHEALTHY,
					LastContact:           ts3,
				},
			},
			skipIndex: false,
		},
		{
			name:      "no previous health status exists",
			oldHealth: &storage.ClusterHealthStatus{},
			newHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        ts1,
			},
			cluster: &storage.Cluster{
				Id: "6",
				HealthStatus: &storage.ClusterHealthStatus{
					SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
					LastContact:        ts1,
				},
			},
			skipIndex: false,
		},
	}

	for _, c := range cases {
		suite.healthStatuses.EXPECT().Get(c.cluster.GetId()).Return(c.oldHealth, true, nil)
		suite.healthStatuses.EXPECT().UpsertWithID(c.cluster.GetId(), c.newHealth)
		if !c.skipIndex {
			suite.clusters.EXPECT().Get(c.cluster.GetId()).Return(c.cluster, true, nil)
			cluster := c.cluster
			cluster.HealthStatus = c.newHealth
			suite.indexer.EXPECT().AddCluster(cluster).Return(nil)
		}

		err := suite.clusterDataStore.UpdateClusterHealth(suite.hasWriteCtx, c.cluster.GetId(), c.newHealth)
		suite.NoError(err)
	}

	err := suite.clusterDataStore.UpdateClusterHealth(suite.hasWriteCtx, "", &storage.ClusterHealthStatus{})
	suite.Error(err)
}
