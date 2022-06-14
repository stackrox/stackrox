package datastore

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	alertMocks "github.com/stackrox/stackrox/central/alert/datastore/mocks"
	clusterIndexMocks "github.com/stackrox/stackrox/central/cluster/index/mocks"
	clusterStoreMocks "github.com/stackrox/stackrox/central/cluster/store/cluster/mocks"
	clusterHealthStoreMocks "github.com/stackrox/stackrox/central/cluster/store/clusterhealth/mocks"
	deploymentMocks "github.com/stackrox/stackrox/central/deployment/datastore/mocks"
	namespaceMocks "github.com/stackrox/stackrox/central/namespace/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/stackrox/central/networkbaseline/manager/mocks"
	netEntityMocks "github.com/stackrox/stackrox/central/networkgraph/entity/datastore/mocks"
	netFlowsMocks "github.com/stackrox/stackrox/central/networkgraph/flow/datastore/mocks"
	nodeMocks "github.com/stackrox/stackrox/central/node/globaldatastore/mocks"
	notifierMocks "github.com/stackrox/stackrox/central/notifier/processor/mocks"
	podMocks "github.com/stackrox/stackrox/central/pod/datastore/mocks"
	"github.com/stackrox/stackrox/central/ranking"
	roleMocks "github.com/stackrox/stackrox/central/rbac/k8srole/datastore/mocks"
	roleBindingMocks "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore/mocks"
	riskMocks "github.com/stackrox/stackrox/central/risk/datastore/mocks"
	"github.com/stackrox/stackrox/central/role/resources"
	secretMocks "github.com/stackrox/stackrox/central/secret/datastore/mocks"
	connectionMocks "github.com/stackrox/stackrox/central/sensor/service/connection/mocks"
	serviceAccountMocks "github.com/stackrox/stackrox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/stackrox/pkg/concurrency"
	graphMocks "github.com/stackrox/stackrox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/images/defaults"
	"github.com/stackrox/stackrox/pkg/protoconv"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stackrox/stackrox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	fakeClusterID   = "FAKECLUSTERID"
	mainImage       = "docker.io/stackrox/rox:latest"
	centralEndpoint = "central.stackrox:443"
)

func TestClusterDataStore(t *testing.T) {
	suite.Run(t, new(ClusterDataStoreTestSuite))
}

type ClusterDataStoreTestSuite struct {
	suite.Suite

	ei          *envisolator.EnvIsolator
	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	clusters                *clusterStoreMocks.MockStore
	healthStatuses          *clusterHealthStoreMocks.MockStore
	indexer                 *clusterIndexMocks.MockIndexer
	clusterDataStore        DataStore
	namespaceDataStore      *namespaceMocks.MockDataStore
	deploymentDataStore     *deploymentMocks.MockDataStore
	nodeDataStore           *nodeMocks.MockGlobalDataStore
	secretDataStore         *secretMocks.MockDataStore
	podDataStore            *podMocks.MockDataStore
	flowsDataStore          *netFlowsMocks.MockClusterDataStore
	netEntityDataStore      *netEntityMocks.MockEntityDataStore
	connMgr                 *connectionMocks.MockManager
	alertDataStore          *alertMocks.MockDataStore
	riskDataStore           *riskMocks.MockDataStore
	mockCtrl                *gomock.Controller
	notifierMock            *notifierMocks.MockProcessor
	mockProvider            *graphMocks.MockProvider
	networkBaselineMgr      *networkBaselineMocks.MockManager
	serviceAccountDataStore *serviceAccountMocks.MockDataStore
	roleDataStore           *roleMocks.MockDataStore
	roleBindingDataStore    *roleBindingMocks.MockDataStore
}

var _ suite.TearDownTestSuite = (*ClusterDataStoreTestSuite)(nil)

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
	suite.clusters = clusterStoreMocks.NewMockStore(suite.mockCtrl)
	suite.healthStatuses = clusterHealthStoreMocks.NewMockStore(suite.mockCtrl)
	suite.indexer = clusterIndexMocks.NewMockIndexer(suite.mockCtrl)

	suite.namespaceDataStore = namespaceMocks.NewMockDataStore(suite.mockCtrl)
	suite.deploymentDataStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.alertDataStore = alertMocks.NewMockDataStore(suite.mockCtrl)
	suite.nodeDataStore = nodeMocks.NewMockGlobalDataStore(suite.mockCtrl)
	suite.secretDataStore = secretMocks.NewMockDataStore(suite.mockCtrl)
	suite.podDataStore = podMocks.NewMockDataStore(suite.mockCtrl)
	suite.flowsDataStore = netFlowsMocks.NewMockClusterDataStore(suite.mockCtrl)
	suite.netEntityDataStore = netEntityMocks.NewMockEntityDataStore(suite.mockCtrl)
	suite.riskDataStore = riskMocks.NewMockDataStore(suite.mockCtrl)
	suite.connMgr = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.notifierMock = notifierMocks.NewMockProcessor(suite.mockCtrl)
	suite.mockProvider = graphMocks.NewMockProvider(suite.mockCtrl)
	suite.networkBaselineMgr = networkBaselineMocks.NewMockManager(suite.mockCtrl)
	suite.serviceAccountDataStore = serviceAccountMocks.NewMockDataStore(suite.mockCtrl)
	suite.roleDataStore = roleMocks.NewMockDataStore(suite.mockCtrl)
	suite.roleBindingDataStore = roleBindingMocks.NewMockDataStore(suite.mockCtrl)

	suite.nodeDataStore.EXPECT().GetAllClusterNodeStores(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	suite.clusters.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
	suite.netEntityDataStore.EXPECT().RegisterCluster(gomock.Any(), gomock.Any()).AnyTimes()
	suite.clusters.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
	suite.healthStatuses.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
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
		suite.podDataStore,
		suite.secretDataStore,
		suite.flowsDataStore,
		suite.netEntityDataStore,
		suite.serviceAccountDataStore,
		suite.roleDataStore,
		suite.roleBindingDataStore,
		suite.connMgr,
		suite.notifierMock,
		suite.mockProvider,
		ranking.NewRanker(),
		suite.networkBaselineMgr,
	)
	suite.NoError(err)
	suite.ei = envisolator.NewEnvIsolator(suite.T())
	suite.ei.Setenv("ROX_IMAGE_FLAVOR", "rhacs")
	testbuildinfo.SetForTest(suite.T())
	testutils.SetExampleVersion(suite.T())
}

func (suite *ClusterDataStoreTestSuite) TearDownTest() {
	suite.ei.RestoreAll()
	suite.mockCtrl.Finish()
}

// Test that when the cluster we try to remove does not exist, we return an error.
func (suite *ClusterDataStoreTestSuite) TestHandlesClusterDoesNotExist() {
	// Return false for the cluster not existing.
	suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return((*storage.Cluster)(nil), false, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID, nil)
	suite.Error(err, "expected an error since the cluster did not exist")
}

// Test that when we cannot fetch a cluster, we return the error from the DB.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingCluster() {
	// Return an error trying to fetch the cluster.
	expectedErr := errors.New("issues need tissues")
	suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return((*storage.Cluster)(nil), true, expectedErr)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID, nil)
	suite.Equal(expectedErr, err)
}

func (suite *ClusterDataStoreTestSuite) TestRemoveCluster() {
	testCluster := &storage.Cluster{Id: fakeClusterID}
	testDeployments := []search.Result{{ID: "fakeDeployment"}}
	testPods := []search.Result{{ID: "fakepod"}}
	testAlerts := []*storage.Alert{{}}
	testSecrets := []*storage.ListSecret{{}}
	testServiceAccounts := []search.Result{{ID: "fakeSA"}}
	testRoles := []search.Result{{ID: "fakeK8Srole"}}
	testRoleBindings := []search.Result{{ID: "fakerolebinding"}}
	suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return(testCluster, true, nil)
	suite.clusters.EXPECT().Delete(suite.hasWriteCtx, fakeClusterID).Return(nil)
	suite.indexer.EXPECT().DeleteCluster(fakeClusterID).Return(nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
	suite.namespaceDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil)
	suite.deploymentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(testDeployments, nil)
	suite.podDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(testPods, nil)
	suite.alertDataStore.EXPECT().SearchRawAlerts(gomock.Any(), gomock.Any()).Return(testAlerts, nil)
	suite.alertDataStore.EXPECT().MarkAlertStale(gomock.Any(), gomock.Any()).Return(nil)
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), gomock.Any()).Return()
	suite.deploymentDataStore.EXPECT().RemoveDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	suite.podDataStore.EXPECT().RemovePod(gomock.Any(), "fakepod").Return(nil)
	suite.nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	suite.secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(testSecrets, nil)
	suite.serviceAccountDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(testServiceAccounts, nil)
	suite.roleDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(testRoles, nil)
	suite.roleBindingDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(testRoleBindings, nil)
	suite.netEntityDataStore.EXPECT().DeleteExternalNetworkEntitiesForCluster(gomock.Any(), fakeClusterID).Return(nil)
	suite.networkBaselineMgr.EXPECT().ProcessPostClusterDelete(gomock.Any()).Return(nil)
	suite.secretDataStore.EXPECT().RemoveSecret(gomock.Any(), gomock.Any()).Return(nil)
	suite.serviceAccountDataStore.EXPECT().RemoveServiceAccount(gomock.Any(), gomock.Any()).Return(nil)
	suite.roleDataStore.EXPECT().RemoveRole(gomock.Any(), gomock.Any()).Return(nil)
	suite.roleBindingDataStore.EXPECT().RemoveRoleBinding(gomock.Any(), gomock.Any()).Return(nil)

	done := concurrency.NewSignal()
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID, &done)
	suite.NoError(err)
	suite.True(concurrency.WaitWithTimeout(&done, 10*time.Second))
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesGet() {
	testCluster := &storage.Cluster{Id: fakeClusterID}
	suite.clusters.EXPECT().Get(gomock.Any(), fakeClusterID).Return(testCluster, true, nil)

	cluster, exists, err := suite.clusterDataStore.GetCluster(suite.hasNoneCtx, fakeClusterID)
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists)
	suite.Nil(cluster, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsGet() {
	suite.clusters.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil)

	_, _, err := suite.clusterDataStore.GetCluster(suite.hasReadCtx, "An Id")
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil)

	_, _, err = suite.clusterDataStore.GetCluster(suite.hasWriteCtx, "beef")
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesGetAll() {
	if features.PostgresDatastore.Enabled() {
		suite.T().Skip("Skipping enforces get all test in postgres mode")
	}
	suite.clusters.EXPECT().GetMany(gomock.Any(), []string{}).Return(nil, nil, nil)
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]*storage.ClusterHealthStatus{}, []int{}, nil)

	clusters, err := suite.clusterDataStore.GetClusters(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Empty(clusters, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsGetAll() {
	suite.clusters.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]*storage.ClusterHealthStatus{}, []int{}, nil)

	_, err := suite.clusterDataStore.GetClusters(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]*storage.ClusterHealthStatus{}, []int{}, nil)

	_, err = suite.clusterDataStore.GetClusters(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesCount() {
	if features.PostgresDatastore.Enabled() {
		suite.T().Skip("Skipping search test in postgres mode")
	}
	suite.clusters.EXPECT().Count(suite.hasWriteCtx).Times(0)
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	count, err := suite.clusterDataStore.CountClusters(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return 0 without access")
	suite.Zero(count, "expected return value to be 0")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsCount() {
	suite.clusters.EXPECT().Count(gomock.Any()).Return(99, nil)

	_, err := suite.clusterDataStore.CountClusters(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().Count(gomock.Any()).Return(42, nil)

	_, err = suite.clusterDataStore.CountClusters(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesAdd() {
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, gomock.Any()).Times(0)

	_, err := suite.clusterDataStore.AddCluster(suite.hasNoneCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")

	_, err = suite.clusterDataStore.AddCluster(suite.hasReadCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsAdd() {
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, gomock.Any()).Return(nil)
	suite.indexer.EXPECT().AddCluster(gomock.Any()).Return(nil)
	suite.flowsDataStore.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(suite.mockCtrl), nil)

	_, err := suite.clusterDataStore.AddCluster(suite.hasWriteCtx, &storage.Cluster{Name: "blah", MainImage: mainImage, CentralApiEndpoint: centralEndpoint})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesUpdate() {
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, gomock.Any()).Times(0)

	err := suite.clusterDataStore.UpdateCluster(suite.hasNoneCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.UpdateCluster(suite.hasReadCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsUpdate() {
	suite.clusters.EXPECT().Get(suite.hasWriteCtx, gomock.Any()).Return(&storage.Cluster{Id: "1", Name: "blah", MainImage: mainImage, CentralApiEndpoint: centralEndpoint}, true, nil)
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, gomock.Any()).Return(nil)
	suite.indexer.EXPECT().AddCluster(gomock.Any()).Return(nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)

	err := suite.clusterDataStore.UpdateCluster(suite.hasWriteCtx, &storage.Cluster{Id: "1", Name: "blah", MainImage: mainImage, CentralApiEndpoint: centralEndpoint})
	suite.NoError(err, "expected no error trying to write with permissions")

	suite.clusters.EXPECT().Get(suite.hasWriteCtx, gomock.Any()).Return(&storage.Cluster{Id: "1", Name: "blah"}, true, nil)

	err = suite.clusterDataStore.UpdateCluster(suite.hasWriteCtx, &storage.Cluster{Id: "1", Name: "blahDiff"})
	suite.Error(err, "expected error trying to rename cluster")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesRemove() {
	suite.clusters.EXPECT().Delete(suite.hasWriteCtx, gomock.Any()).Times(0)

	err := suite.clusterDataStore.RemoveCluster(suite.hasNoneCtx, "jiogserlksd", nil)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.RemoveCluster(suite.hasReadCtx, "vflkjdf", nil)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsRemove() {
	// This is a weird thing for store.Get() to return but we're only testing auth here
	suite.clusters.EXPECT().Get(suite.hasWriteCtx, "poiuytre").Return(nil, true, nil)
	suite.clusters.EXPECT().Delete(suite.hasWriteCtx, gomock.Any()).Return(nil)
	suite.indexer.EXPECT().DeleteCluster(gomock.Any()).Return(nil)
	suite.namespaceDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.deploymentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	suite.netEntityDataStore.EXPECT().DeleteExternalNetworkEntitiesForCluster(gomock.Any(), gomock.Any()).Return(nil)
	suite.networkBaselineMgr.EXPECT().ProcessPostClusterDelete(gomock.Any()).Return(nil)
	suite.secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.serviceAccountDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.roleDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.roleBindingDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.podDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
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
	suite.clusters.EXPECT().Get(suite.hasWriteCtx, gomock.Any()).Return(&storage.Cluster{Id: "qwerty", Name: "blah"}, true, nil)
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, &storage.Cluster{Id: "qwerty", Name: "blah", Status: &storage.ClusterStatus{}}).Return(nil)

	err := suite.clusterDataStore.UpdateClusterStatus(suite.hasWriteCtx, "qwerty", &storage.ClusterStatus{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesSearch() {
	if features.PostgresDatastore.Enabled() {
		suite.T().Skip("Skipping search test in postgres mode")
	}
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
	suite.clusters.EXPECT().GetMany(gomock.Any(), ids).Return(clusters, []int{}, nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any(), ids).Return(existingHealths, []int{0, 2, 5}, nil)

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
	suite.clusters.EXPECT().GetMany(gomock.Any(), ids).Return(clusters, []int{}, nil)
	suite.healthStatuses.EXPECT().GetMany(gomock.Any(), ids).Return(existingHealths, []int{}, nil)

	actuals, err = suite.clusterDataStore.SearchRawClusters(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Equal(expected, actuals)
}

func (suite *ClusterDataStoreTestSuite) TestLookupOrCreateClusterFromConfig() {
	const bundleID = "aW5pdC1idW5kbGUtaWQK"
	const policyVersion = "1"
	const someHelmConfigJSON = `{
		"staticConfig": {
		  "type": "KUBERNETES_CLUSTER",
		  "mainImage": "docker.io/stackrox/main",
		  "centralApiEndpoint": "central.stackrox.svc:443",
		  "collectorImage": "docker.io/stackrox/collector"
		},
		"configFingerprint": "1234"
	  }`
	const differentConfigFPHelmConfigJSON = `{
		"staticConfig": {
		  "type": "KUBERNETES_CLUSTER",
		  "mainImage": "docker.io/stackrox/main",
		  "centralApiEndpoint": "central.stackrox.svc:443",
		  "collectorImage": "docker.io/stackrox/collector"
		},
		"configFingerprint": "12345"
	  }`
	var someHelmConfig storage.CompleteClusterConfig
	var differentConfigFPHelmConfig storage.CompleteClusterConfig
	ts := protoconv.ConvertTimeToTimestamp(time.Now())
	clusterHealth := &storage.ClusterHealthStatus{
		SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		LastContact:        ts,
	}

	err := jsonpb.Unmarshal(bytes.NewReader([]byte(someHelmConfigJSON)), &someHelmConfig)
	suite.NoError(err)

	err = jsonpb.Unmarshal(bytes.NewReader([]byte(differentConfigFPHelmConfigJSON)), &differentConfigFPHelmConfig)
	suite.NoError(err)

	someClusterWithManagerType := func(managerType storage.ManagerType, helmConfig *storage.CompleteClusterConfig) *storage.Cluster {
		return &storage.Cluster{
			Id:                 "",
			Name:               "",
			InitBundleId:       bundleID,
			HelmConfig:         helmConfig,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
			ManagedBy:          managerType,
		}
	}

	sensorHelloWithHelmManagedConfigInit := func(helmManagedConfigInit *central.HelmManagedConfigInit) *central.SensorHello {
		return &central.SensorHello{
			DeploymentIdentification: &storage.SensorDeploymentIdentification{},
			HelmManagedConfigInit:    helmManagedConfigInit,
			PolicyVersion:            policyVersion,
		}
	}

	cases := []struct {
		description         string
		cluster             *storage.Cluster
		sensorHello         *central.SensorHello
		bundleID            string
		expectedManagerType storage.ManagerType
		expectedHelmConfig  *storage.CompleteClusterConfig
		expectClusterUpsert bool
	}{
		{
			description: "existing cluster's UNKNOWN manager type unchanged if notHelmManaged=false and managedBy=null",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_UNKNOWN, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				NotHelmManaged: false,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_UNKNOWN,
			expectedHelmConfig:  &someHelmConfig,
		},
		// Test if clusters UNKNOWN manager type can be upgraded to MANUAL/HELM_CHART/KUBERNETES_OPERATOR.
		{
			description: "existing cluster's UNKNOWN manager type can be upgraded to MANUAL if notHelmManaged=true and managedBy=null",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_UNKNOWN, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				NotHelmManaged: true,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:  nil,
			expectClusterUpsert: true,
		},
		{
			description: "existing cluster's UNKNOWN manager type can be upgraded to HELM_CHART",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_UNKNOWN, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				NotHelmManaged: false,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedHelmConfig:  &someHelmConfig,
			expectClusterUpsert: true,
		},
		{
			description: "existing cluster's UNKNOWN manager type can be upgraded to KUBERNETES_OPERATOR",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_UNKNOWN, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
				NotHelmManaged: false,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
			expectedHelmConfig:  &someHelmConfig,
			expectClusterUpsert: true,
		},
		// Test if clusters non-MANUAL manager type can be changed to MANUAL using notHelmManaged=true.
		{
			description: "existing cluster's HELM_CHART manager type can be changed to MANUAL if notHelmManaged=true",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				NotHelmManaged: true,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:  nil,
			expectClusterUpsert: true,
		},
		{
			description: "existing cluster's HELM_CHART manager type can be changed to MANUAL if notHelmManaged=true and managedBy=MANUAL",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_MANUAL,
				NotHelmManaged: true,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:  nil,
			expectClusterUpsert: true,
		},
		{
			description: "existing cluster's KUBERNETES_OPERATOR manager type can be changed to MANUAL if notHelmManaged=true",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				NotHelmManaged: true,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:  nil,
			expectClusterUpsert: true,
		},
		{
			description: "existing cluster's KUBERNETES_OPERATOR manager type can be changed to MANUAL if notHelmManaged=true and managedBy=MANUAL",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_MANUAL,
				NotHelmManaged: true,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:  nil,
			expectClusterUpsert: true,
		},
		// Test if new clusters can be added with desired manager type.
		{
			description: "new cluster with manager type KUBERNETES_OPERATOR can be created",
			cluster:     nil,
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
				NotHelmManaged: false,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
			expectedHelmConfig:  &someHelmConfig,
			expectClusterUpsert: true,
		},
		{
			description: "new cluster with manager type HELM_CHART can be created",
			cluster:     nil,
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				NotHelmManaged: false,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedHelmConfig:  &someHelmConfig,
			expectClusterUpsert: true,
		},
		{
			description: "new cluster with manager type MANUAL can be created",
			cluster:     nil,
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_MANUAL,
				NotHelmManaged: true,
				ClusterConfig:  &someHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:  nil,
			expectClusterUpsert: true,
		},
		// Updating Helm configuration
		{
			description: "existing cluster's Helm configuration can be updated for Helm-managed clusters",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:      storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				NotHelmManaged: false,
				ClusterConfig:  &differentConfigFPHelmConfig,
			}),
			bundleID:            bundleID,
			expectedManagerType: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedHelmConfig:  &differentConfigFPHelmConfig,
			expectClusterUpsert: true,
		},
	}

	for i, c := range cases {
		suite.T().Run(c.description, func(t *testing.T) {
			var clusterID string
			var newCluster *storage.Cluster

			// Make cluster name unique to simplify testing code due to caching.
			clusterName := fmt.Sprintf("test_lookup_or_create_%d", i)
			if c.cluster != nil {
				c.cluster.Name = clusterName
			}
			if helmCfg := c.sensorHello.GetHelmManagedConfigInit(); helmCfg != nil {
				helmCfg.ClusterName = clusterName
			}

			addedClusterMock := suite.indexer.EXPECT().AddCluster(gomock.Any()).Do(func(updatedCluster *storage.Cluster) {
				clusterID = updatedCluster.GetId()
				newCluster = updatedCluster
			}).Return(nil)

			upsertMock := suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, gomock.Any()).Do(func(_ context.Context, updatedCluster *storage.Cluster) {
				clusterID = updatedCluster.GetId()
				newCluster = updatedCluster
			}).Return(nil)

			if c.expectClusterUpsert {
				addedClusterMock.Times(2)
				upsertMock.Times(2)
			} else {
				addedClusterMock.Times(1)
				upsertMock.Times(1)
			}

			suite.flowsDataStore.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(suite.mockCtrl), nil)
			if cluster := c.cluster; cluster != nil {
				clusterID, err = suite.clusterDataStore.AddCluster(suite.hasWriteCtx, cluster)
				suite.NoError(err)

				suite.clusters.EXPECT().Get(suite.hasWriteCtx, clusterID).Return(newCluster, true, nil)
				suite.healthStatuses.EXPECT().GetMany(suite.hasWriteCtx, []string{clusterID}).Return([]*storage.ClusterHealthStatus{clusterHealth}, []int{}, nil)
			}

			// Execute call to LookupOrCreateClusterFromConfig.
			_, err = suite.clusterDataStore.LookupOrCreateClusterFromConfig(suite.hasWriteCtx, clusterID, c.bundleID, c.sensorHello)
			suite.NoError(err)

			// Check results.
			suite.Equal(storage.ManagerType_name[int32(c.expectedManagerType)], storage.ManagerType_name[int32(newCluster.GetManagedBy())])
			suite.Equal(c.expectedHelmConfig, newCluster.GetHelmConfig())
		})
	}
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
				Id:                 "6",
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
		suite.healthStatuses.EXPECT().Get(suite.hasWriteCtx, c.cluster.GetId()).Return(c.oldHealth, true, nil)
		suite.healthStatuses.EXPECT().Upsert(suite.hasWriteCtx, c.newHealth)
		if !c.skipIndex {
			suite.clusters.EXPECT().Get(suite.hasWriteCtx, c.cluster.GetId()).Return(c.cluster, true, nil)
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

func (suite *ClusterDataStoreTestSuite) TestUpdateAuditLogFileStates() {
	t1 := time.Now()
	t2 := time.Now().Add(-30 * time.Minute)
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts2 := protoconv.ConvertTimeToTimestamp(t2)

	fakeCluster := &storage.Cluster{Id: fakeClusterID, Name: "it's just your imagination"}

	states := map[string]*storage.AuditLogFileState{
		"node-1": {CollectLogsSince: ts1, LastAuditId: "abcd"},
		"node-2": {CollectLogsSince: ts2, LastAuditId: "efgh"},
		"node-3": {CollectLogsSince: ts1, LastAuditId: "zyxw"},
	}

	suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return(fakeCluster, true, nil)
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, &storage.Cluster{Id: fakeClusterID, Name: "it's just your imagination", AuditLogState: states}).Return(nil)

	err := suite.clusterDataStore.UpdateAuditLogFileStates(suite.hasWriteCtx, fakeClusterID, states)
	suite.NoError(err)
}

func (suite *ClusterDataStoreTestSuite) TestUpdateAuditLogFileStatesLeavesUnmodifiedNodesAlone() {
	t1 := time.Now()
	t2 := time.Now().Add(-30 * time.Minute)
	t3 := time.Now().Add(-10 * time.Minute)
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts2 := protoconv.ConvertTimeToTimestamp(t2)
	ts3 := protoconv.ConvertTimeToTimestamp(t3)

	fakeCluster := &storage.Cluster{
		Id:   fakeClusterID,
		Name: "it's just your imagination",
		AuditLogState: map[string]*storage.AuditLogFileState{
			"old-node1": {CollectLogsSince: ts3, LastAuditId: "ggggg"},
		},
	}

	newStates := map[string]*storage.AuditLogFileState{
		"node-1": {CollectLogsSince: ts1, LastAuditId: "abcd"},
		"node-2": {CollectLogsSince: ts2, LastAuditId: "efgh"},
	}

	expectedStates := map[string]*storage.AuditLogFileState{
		"node-1":    {CollectLogsSince: ts1, LastAuditId: "abcd"},
		"node-2":    {CollectLogsSince: ts2, LastAuditId: "efgh"},
		"old-node1": {CollectLogsSince: ts3, LastAuditId: "ggggg"},
	}

	suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return(fakeCluster, true, nil)
	suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, &storage.Cluster{Id: fakeClusterID, Name: "it's just your imagination", AuditLogState: expectedStates}).Return(nil)

	err := suite.clusterDataStore.UpdateAuditLogFileStates(suite.hasWriteCtx, fakeClusterID, newStates)
	suite.NoError(err)
}

func (suite *ClusterDataStoreTestSuite) TestUpdateAuditLogFileStatesErrorConditions() {
	t1 := time.Now()
	t2 := time.Now().Add(-30 * time.Minute)
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts2 := protoconv.ConvertTimeToTimestamp(t2)

	fakeCluster := &storage.Cluster{Id: fakeClusterID, Name: "it's just your imagination"}

	states := map[string]*storage.AuditLogFileState{
		"node-1": {CollectLogsSince: ts1, LastAuditId: "abcd"},
		"node-2": {CollectLogsSince: ts2, LastAuditId: "efgh"},
		"node-3": {CollectLogsSince: ts1, LastAuditId: "zyxw"},
	}

	cases := []struct {
		name             string
		ctx              context.Context
		clusterID        string
		states           map[string]*storage.AuditLogFileState
		clusterIsMissing bool
		realClusterFound bool
		upsertWillError  bool
	}{
		{
			name:             "Error when no cluster id is provided",
			ctx:              suite.hasWriteCtx,
			clusterID:        "",
			states:           states,
			clusterIsMissing: false,
			upsertWillError:  false,
		},
		{
			name:             "Error when no states are provided",
			ctx:              suite.hasWriteCtx,
			clusterID:        fakeClusterID,
			states:           nil,
			clusterIsMissing: false,
			upsertWillError:  false,
		},
		{
			name:             "Error when empty states are provided",
			ctx:              suite.hasWriteCtx,
			clusterID:        fakeClusterID,
			states:           map[string]*storage.AuditLogFileState{},
			clusterIsMissing: false,
			upsertWillError:  false,
		},
		{
			name:             "Error when context has no perms",
			ctx:              suite.hasNoneCtx,
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: false,
			upsertWillError:  false,
		},
		{
			name:             "Error when is not read only",
			ctx:              suite.hasReadCtx,
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: false,
			realClusterFound: false,
			upsertWillError:  false,
		},
		{
			name:             "Error when cluster cannot be found",
			ctx:              suite.hasWriteCtx,
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: true,
			realClusterFound: false,
			upsertWillError:  false,
		},
		{
			name:             "Error when Upsert fails",
			ctx:              suite.hasWriteCtx,
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: false,
			realClusterFound: true,
			upsertWillError:  true,
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			if c.clusterIsMissing {
				suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return((*storage.Cluster)(nil), false, nil)
			}
			if c.realClusterFound {
				suite.clusters.EXPECT().Get(suite.hasWriteCtx, fakeClusterID).Return(fakeCluster, true, nil)
			}
			if c.upsertWillError {
				suite.clusters.EXPECT().Upsert(suite.hasWriteCtx, &storage.Cluster{Id: fakeClusterID, Name: "it's just your imagination", AuditLogState: states}).Return(errors.New("test"))
			}
			err := suite.clusterDataStore.UpdateAuditLogFileStates(c.ctx, c.clusterID, c.states)
			suite.Error(err)
		})
	}
}

func (suite *ClusterDataStoreTestSuite) TestNormalizeCluster() {
	cases := []struct {
		name     string
		cluster  *storage.Cluster
		expected string
	}{
		{
			name: "Happy path",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "localhost:8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "http",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "http://localhost:8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "https",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "https://localhost:8080",
			},
			expected: "localhost:8080",
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			suite.NoError(normalizeCluster(c.cluster))
			suite.Equal(c.expected, c.cluster.GetCentralApiEndpoint())
		})
	}
}

func (suite *ClusterDataStoreTestSuite) TestValidateCluster() {
	cases := []struct {
		name          string
		cluster       *storage.Cluster
		expectedError bool
	}{
		{
			name:          "Empty Cluster",
			cluster:       &storage.Cluster{},
			expectedError: true,
		},
		{
			name: "No name",
			cluster: &storage.Cluster{
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Image",
			cluster: &storage.Cluster{
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "Image without tag",
			cluster: &storage.Cluster{
				MainImage:          "stackrox/main",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Non-trivial image",
			cluster: &storage.Cluster{
				MainImage:          "stackrox/main:1.2",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Moderately complex image",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main:1.2.512-125125",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Image with SHA",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Invalid image - contains spaces",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main:1.2.3 injectedCommand",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Central Endpoint",
			cluster: &storage.Cluster{
				Name:      "name",
				MainImage: "image",
			},
			expectedError: true,
		},
		{
			name: "Central Endpoint w/o port",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central",
			},
			expectedError: true,
		},
		{
			name: "Valid collector registry",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				CollectorImage:     "collector.stackrox.io/collector",
			},
			expectedError: false,
		},
		{
			name: "Empty string collector registry",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				CollectorImage:     "",
			},
		},
		{
			name: "Invalid collector registry",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				CollectorImage:     "collector.stackrox.io/collector injectedCommand",
			},
			expectedError: true,
		},
		{
			name: "Happy path K8s",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Happy path",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			cluster := c.cluster.Clone()
			cluster.DynamicConfig = &storage.DynamicClusterConfig{
				DisableAuditLogs: true,
			}
			err := validateInput(cluster)
			if c.expectedError {
				suite.Error(err)
			} else {
				suite.Nil(err)
			}
		})
	}

}

func (suite *ClusterDataStoreTestSuite) TestAddDefaults() {

	suite.Run("Error on nil cluster", func() {
		suite.Error(addDefaults(nil))
	})

	flavor := defaults.GetImageFlavorFromEnv()
	suite.Run("Some default values are set for uninialized fields", func() {
		cluster := &storage.Cluster{}
		suite.NoError(addDefaults(cluster))
		suite.Equal(flavor.MainImageNoTag(), cluster.GetMainImage())
		suite.Empty(cluster.GetCollectorImage()) // must not be set
		suite.Equal(centralEndpoint, cluster.GetCentralApiEndpoint())
		suite.True(cluster.GetRuntimeSupport())
		suite.Equal(storage.CollectionMethod_KERNEL_MODULE, cluster.GetCollectionMethod())
		if tc := cluster.GetTolerationsConfig(); suite.NotNil(tc) {
			suite.False(tc.GetDisabled())
		}
		if dc := cluster.GetDynamicConfig(); suite.NotNil(dc) {
			suite.True(dc.GetDisableAuditLogs())
			if acc := dc.GetAdmissionControllerConfig(); suite.NotNil(acc) {
				suite.False(acc.GetEnabled())
				suite.Equal(int32(defaultAdmissionControllerTimeout),
					acc.GetTimeoutSeconds())
			}
		}
	})

	suite.Run("Provided values are either not overridden or properly updated", func() {
		cluster := &storage.Cluster{
			Id:                         fakeClusterID,
			Name:                       "someName",
			Type:                       storage.ClusterType_KUBERNETES_CLUSTER,
			Labels:                     map[string]string{"key": "value"},
			MainImage:                  "somevalue",
			CollectorImage:             "someOtherValue",
			CentralApiEndpoint:         "someEndpoint",
			RuntimeSupport:             true,
			CollectionMethod:           storage.CollectionMethod_EBPF,
			AdmissionController:        true,
			AdmissionControllerUpdates: true,
			AdmissionControllerEvents:  true,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					Enabled:        true,
					TimeoutSeconds: 73,
				},
				RegistryOverride: "registryOverride",
				DisableAuditLogs: false,
			},
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: true,
			},
			Priority:      10,
			SlimCollector: true,
			HelmConfig:    &storage.CompleteClusterConfig{},
			InitBundleId:  "someId",
			ManagedBy:     storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
		}
		suite.NoError(addDefaults(cluster))

		suite.Equal(fakeClusterID, cluster.GetId())
		suite.Equal("someName", cluster.GetName())
		suite.Equal(storage.ClusterType_KUBERNETES_CLUSTER, cluster.GetType())
		suite.EqualValues(map[string]string{"key": "value"}, cluster.GetLabels())

		suite.Equal("somevalue", cluster.GetMainImage())
		suite.Equal("someOtherValue", cluster.GetCollectorImage())
		suite.Equal("someEndpoint", cluster.GetCentralApiEndpoint())
		suite.True(cluster.GetRuntimeSupport())
		suite.Equal(storage.CollectionMethod_EBPF, cluster.GetCollectionMethod())
		suite.True(cluster.GetAdmissionController())
		suite.True(cluster.GetAdmissionControllerUpdates())
		suite.True(cluster.GetAdmissionControllerEvents())
		if dc := cluster.GetDynamicConfig(); suite.NotNil(dc) {
			suite.Equal("registryOverride", dc.GetRegistryOverride())
			suite.True(dc.GetDisableAuditLogs()) // True for KUBERNETES_CLUSTER
			if acc := dc.GetAdmissionControllerConfig(); suite.NotNil(acc) {
				suite.True(acc.GetEnabled())
				suite.Equal(int32(73), acc.GetTimeoutSeconds())
			}
		}
		if tc := cluster.GetTolerationsConfig(); suite.NotNil(tc) {
			suite.True(tc.GetDisabled())
		}
		suite.Equal(int64(10), cluster.GetPriority())
		suite.True(cluster.SlimCollector)
		suite.NotNil(cluster.GetHelmConfig())
		suite.Equal("someId", cluster.GetInitBundleId())
		suite.Equal(storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR, cluster.GetManagedBy())
	})

	suite.Run("Audit logs", func() {
		for name, testCase := range map[string]struct {
			cluster              *storage.Cluster
			expectedDisabledLogs bool
		}{
			"Kubernetes cluster":  {&storage.Cluster{Type: storage.ClusterType_KUBERNETES_CLUSTER}, true},
			"Openshift 3 cluster": {&storage.Cluster{Type: storage.ClusterType_OPENSHIFT_CLUSTER}, true},
			"Openshift 4 cluster": {&storage.Cluster{Type: storage.ClusterType_OPENSHIFT4_CLUSTER}, false},
			"Openshift 4 cluster with disabled logs": {&storage.Cluster{Type: storage.ClusterType_OPENSHIFT4_CLUSTER,
				DynamicConfig: &storage.DynamicClusterConfig{DisableAuditLogs: true}}, true},
		} {
			suite.Run(name, func() {
				suite.NoError(addDefaults(testCase.cluster))
				if dc := testCase.cluster.GetDynamicConfig(); suite.NotNil(dc) {
					suite.Equal(testCase.expectedDisabledLogs, dc.GetDisableAuditLogs())
				}
			})
		}
	})

	suite.Run("Collector image not set when only main image is provided", func() {
		cluster := &storage.Cluster{
			MainImage: "somevalue",
		}
		suite.NoError(addDefaults(cluster))
		suite.Empty(cluster.GetCollectorImage())
	})

	suite.Run("Error for bad timeout", func() {
		cluster := &storage.Cluster{
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					TimeoutSeconds: -1,
				}},
		}
		suite.Error(addDefaults(cluster))
	})

	for method, runtimeSupport := range map[storage.CollectionMethod]bool{
		storage.CollectionMethod_UNSET_COLLECTION: true,
		storage.CollectionMethod_NO_COLLECTION:    false,
		storage.CollectionMethod_KERNEL_MODULE:    true,
		storage.CollectionMethod_EBPF:             true,
	} {
		suite.Run(fmt.Sprintf("Runtime support for %s collection method", method), func() {
			cluster := &storage.Cluster{
				CollectionMethod: method,
			}
			suite.NoError(addDefaults(cluster))
			suite.Equal(runtimeSupport, cluster.GetRuntimeSupport())
		})
	}
}
