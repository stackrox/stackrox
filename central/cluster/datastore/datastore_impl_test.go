package datastore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterIndexMocks "github.com/stackrox/rox/central/cluster/index/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/store/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/processor/mocks"
	"github.com/stackrox/rox/central/role/resources"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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

	clusters            *clusterMocks.MockStore
	indexer             *clusterIndexMocks.MockIndexer
	clusterDataStore    DataStore
	deploymentDataStore *deploymentMocks.MockDataStore
	nodeDataStore       *nodeMocks.MockGlobalDataStore
	secretDataStore     *secretMocks.MockDataStore
	connMgr             *connectionMocks.MockManager
	alertDataStore      *alertMocks.MockDataStore

	mockCtrl     *gomock.Controller
	notifierMock *notifierMocks.MockProcessor
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
	suite.clusters = clusterMocks.NewMockStore(suite.mockCtrl)
	suite.indexer = clusterIndexMocks.NewMockIndexer(suite.mockCtrl)

	suite.deploymentDataStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.alertDataStore = alertMocks.NewMockDataStore(suite.mockCtrl)
	suite.nodeDataStore = nodeMocks.NewMockGlobalDataStore(suite.mockCtrl)
	suite.secretDataStore = secretMocks.NewMockDataStore(suite.mockCtrl)
	suite.connMgr = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.notifierMock = notifierMocks.NewMockProcessor(suite.mockCtrl)

	suite.nodeDataStore.EXPECT().GetAllClusterNodeStores(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	suite.clusters.EXPECT().GetClusters().Return(([]*storage.Cluster)(nil), nil)
	suite.indexer.EXPECT().AddClusters(nil).Return(nil)

	var err error
	suite.clusterDataStore, err = New(suite.clusters, suite.indexer, suite.alertDataStore, suite.deploymentDataStore, suite.nodeDataStore, suite.secretDataStore, suite.connMgr, suite.notifierMock)
	suite.NoError(err)
}

func (suite *ClusterDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

//// Test that when the cluster we try to remove does not exist, we return an error.
func (suite *ClusterDataStoreTestSuite) TestHandlesClusterDoesNotExist() {
	// Return false for the cluster not existing.
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return((*storage.Cluster)(nil), false, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID)
	suite.Error(err, "expected an error since the cluster did not exist")
}

// Test that when we cannot fetch a cluster, we return the error from the DB.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingCluster() {
	// Return an error trying to fetch the cluster.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return((*storage.Cluster)(nil), true, expectedErr)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID)
	suite.Equal(expectedErr, err)
}

func (suite *ClusterDataStoreTestSuite) TestRemoveCluster() {
	testCluster := &storage.Cluster{Id: fakeClusterID}
	testDeployments := []*storage.ListDeployment{{ClusterId: fakeClusterID}}
	testAlerts := []*storage.Alert{{}}
	testSecrets := []*storage.ListSecret{{}}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(testCluster, true, nil)
	suite.clusters.EXPECT().RemoveCluster(fakeClusterID).Return(nil)
	suite.indexer.EXPECT().DeleteCluster(fakeClusterID).Return(nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
	suite.deploymentDataStore.EXPECT().ListDeployments(gomock.Any()).Return(testDeployments, nil)
	suite.alertDataStore.EXPECT().SearchRawAlerts(gomock.Any(), gomock.Any()).Return(testAlerts, nil)
	suite.alertDataStore.EXPECT().MarkAlertStale(gomock.Any(), gomock.Any()).Return(nil)
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any()).Return()
	suite.deploymentDataStore.EXPECT().RemoveDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	suite.nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	suite.secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(testSecrets, nil)
	suite.secretDataStore.EXPECT().RemoveSecret(gomock.Any(), gomock.Any()).Return(nil)

	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, fakeClusterID)
	suite.NoError(err)
	time.Sleep(200000)
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesGet() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().GetCluster(gomock.Any()).Times(0)

	cluster, exists, err := suite.clusterDataStore.GetCluster(suite.hasNoneCtx, "hkjddjhk")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists)
	suite.Nil(cluster, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsGet() {
	suite.clusters.EXPECT().GetCluster(gomock.Any()).Return(nil, false, nil)

	_, _, err := suite.clusterDataStore.GetCluster(suite.hasReadCtx, "An Id")
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().GetCluster(gomock.Any()).Return(nil, false, nil)

	_, _, err = suite.clusterDataStore.GetCluster(suite.hasWriteCtx, "beef")
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesGetAll() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().GetClusters().Times(0)
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	clusters, err := suite.clusterDataStore.GetClusters(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Empty(clusters, "expected return value to be nil")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsGetAll() {
	suite.clusters.EXPECT().GetClusters().Return(nil, nil)

	_, err := suite.clusterDataStore.GetClusters(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().GetClusters().Return(nil, nil)

	_, err = suite.clusterDataStore.GetClusters(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesCount() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().CountClusters().Times(0)
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	count, err := suite.clusterDataStore.CountClusters(suite.hasNoneCtx)
	suite.NoError(err, "expected no error, should return 0 without access")
	suite.Zero(count, "expected return value to be 0")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsCount() {
	suite.clusters.EXPECT().CountClusters().Return(99, nil)

	_, err := suite.clusterDataStore.CountClusters(suite.hasReadCtx)
	suite.NoError(err, "expected no error trying to read with permissions")

	suite.clusters.EXPECT().CountClusters().Return(42, nil)

	_, err = suite.clusterDataStore.CountClusters(suite.hasWriteCtx)
	suite.NoError(err, "expected no error trying to read with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesAdd() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().AddCluster(gomock.Any()).Times(0)

	_, err := suite.clusterDataStore.AddCluster(suite.hasNoneCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")

	_, err = suite.clusterDataStore.AddCluster(suite.hasReadCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsAdd() {
	suite.clusters.EXPECT().AddCluster(gomock.Any()).Return("hsdhjkbf", nil)
	suite.indexer.EXPECT().AddCluster(gomock.Any()).Return(nil)

	_, err := suite.clusterDataStore.AddCluster(suite.hasWriteCtx, &storage.Cluster{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesUpdate() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().UpdateCluster(gomock.Any()).Times(0)

	err := suite.clusterDataStore.UpdateCluster(suite.hasNoneCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.UpdateCluster(suite.hasReadCtx, &storage.Cluster{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsUpdate() {
	suite.clusters.EXPECT().UpdateCluster(gomock.Any()).Return(nil)
	suite.indexer.EXPECT().AddCluster(gomock.Any()).Return(nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)

	err := suite.clusterDataStore.UpdateCluster(suite.hasWriteCtx, &storage.Cluster{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesRemove() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().RemoveCluster(gomock.Any()).Times(0)

	err := suite.clusterDataStore.RemoveCluster(suite.hasNoneCtx, "jiogserlksd")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.RemoveCluster(suite.hasReadCtx, "vflkjdf")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsRemove() {
	// This is a weird thing for store.GetCluster() to return but we're only testing auth here
	suite.clusters.EXPECT().GetCluster("poiuytre").Return(nil, true, nil)
	suite.clusters.EXPECT().RemoveCluster(gomock.Any()).Return(nil)
	suite.indexer.EXPECT().DeleteCluster(gomock.Any()).Return(nil)
	suite.deploymentDataStore.EXPECT().ListDeployments(gomock.Any()).Return(nil, nil)
	suite.nodeDataStore.EXPECT().RemoveClusterNodeStores(gomock.Any(), gomock.Any()).Return(nil)
	suite.secretDataStore.EXPECT().SearchListSecrets(gomock.Any(), gomock.Any()).Return(nil, nil)
	suite.connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)

	err := suite.clusterDataStore.RemoveCluster(suite.hasWriteCtx, "poiuytre")
	suite.NoError(err, "expected no error trying to write with permissions")
	// RemoveCluster invokes a goroutine that calls a bunch of mocks.  Wait .05 seconds for this to complete
	time.Sleep(50000)
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesUpdateClusterContactTime() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().UpdateCluster(gomock.Any()).Times(0)

	err := suite.clusterDataStore.UpdateClusterContactTime(suite.hasNoneCtx, "F", time.Now())
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.UpdateClusterContactTime(suite.hasReadCtx, "IDK", time.Now())
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsUpdateClusterContactTime() {
	suite.clusters.EXPECT().UpdateClusterContactTime(gomock.Any(), gomock.Any()).Return(nil)

	err := suite.clusterDataStore.UpdateClusterContactTime(suite.hasWriteCtx, "qwerty", time.Now())
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesUpdateClusterStatus() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	suite.clusters.EXPECT().UpdateCluster(gomock.Any()).Times(0)

	err := suite.clusterDataStore.UpdateClusterStatus(suite.hasNoneCtx, "F", &storage.ClusterStatus{})
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.clusterDataStore.UpdateClusterStatus(suite.hasReadCtx, "IDK", &storage.ClusterStatus{})
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ClusterDataStoreTestSuite) TestAllowsUpdateClusterStatus() {
	suite.clusters.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any()).Return(nil)

	err := suite.clusterDataStore.UpdateClusterStatus(suite.hasWriteCtx, "qwerty", &storage.ClusterStatus{})
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ClusterDataStoreTestSuite) TestEnforcesSearch() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
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
