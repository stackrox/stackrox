package datastore

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/store/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/store/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

const fakeClusterID = "FAKECLUSTERID"

func TestClusterDataStore(t *testing.T) {
	suite.Run(t, new(ClusterDataStoreTestSuite))
}

type ClusterDataStoreTestSuite struct {
	suite.Suite

	clusters    *clusterMocks.MockStore
	deployments *deploymentMocks.MockDataStore
	alerts      *alertMocks.MockDataStore
	nodes       *nodeMocks.MockGlobalStore
	secrets     *secretMocks.MockDataStore

	clusterDataStore DataStore

	mockCtrl *gomock.Controller
}

func (suite *ClusterDataStoreTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.clusters = clusterMocks.NewMockStore(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.alerts = alertMocks.NewMockDataStore(suite.mockCtrl)
	suite.nodes = nodeMocks.NewMockGlobalStore(suite.mockCtrl)
	suite.secrets = secretMocks.NewMockDataStore(suite.mockCtrl)

	suite.clusterDataStore = New(suite.clusters, suite.alerts, suite.deployments, suite.nodes, suite.secrets)
}

func (suite *ClusterDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// Test the happy path.
func (suite *ClusterDataStoreTestSuite) TestRemoveTombstonesDeploymentsAndMarksAlertsStale() {
	// We expect alerts to be fetched, and all to be updated.
	alerts := getAlerts(2)
	suite.alerts.EXPECT().ListAlerts(
		testutils.PredMatcher("query for deployment1", func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(alerts, nil)
	for _, alert := range alerts {
		suite.alerts.EXPECT().MarkAlertStale(alert.GetId()).Return(nil)
	}

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": fakeClusterID, "deployment2": "cluster2"})
	suite.deployments.EXPECT().ListDeployments().Return(deployments, nil)
	suite.deployments.EXPECT().RemoveDeployment("deployment1").Return(nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)
	suite.clusters.EXPECT().RemoveCluster(fakeClusterID).Return(nil)

	suite.nodes.EXPECT().RemoveClusterNodeStore(fakeClusterID).Times(1).Return(nil)

	suite.secrets.EXPECT().SearchListSecrets(search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.GetId()).ProtoQuery()).Return([]*v1.ListSecret{}, nil)

	// run removal.
	suite.clusterDataStore.RemoveCluster(fakeClusterID)
}

//// Test that when the cluster we try to remove does not exist, we return an error.
func (suite *ClusterDataStoreTestSuite) TestHandlesClusterDoesNotExist() {
	// Return false for the cluster not existing.
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return((*storage.Cluster)(nil), false, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Error(err, "expected an error since the cluster did not exist")
}

// Test that when we cannot fetch a cluster, we return the error from the DB.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingCluster() {
	// Return an error trying to fetch the cluster.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return((*storage.Cluster)(nil), true, expectedErr)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Equal(expectedErr, err)
}

// Test that when no deployments exist for a cluster, the cluster is removed successfully with no additional
// operations on either deployments or alerts.
func (suite *ClusterDataStoreTestSuite) TestHandlesNoDeployments() {
	// Return an error trying to fetch the deployments for a cluster.
	suite.deployments.EXPECT().ListDeployments().Return(([]*storage.ListDeployment)(nil), nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)
	suite.clusters.EXPECT().RemoveCluster(fakeClusterID).Return(nil)

	suite.nodes.EXPECT().RemoveClusterNodeStore(fakeClusterID).Times(1).Return(nil)

	suite.secrets.EXPECT().SearchListSecrets(search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.GetId()).ProtoQuery()).Return([]*v1.ListSecret{}, nil)

	// run removal.
	suite.clusterDataStore.RemoveCluster(fakeClusterID)
}

// Test that when we get an error trying to fetch the deployments for a cluster, we do not remove the cluster
// and instead return the error.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingDeployments() {
	// Return an error trying to fetch the deployments for a cluster.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.deployments.EXPECT().ListDeployments().Return(([]*storage.ListDeployment)(nil), expectedErr)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Equal(expectedErr, err)
}

// Test that when we are unable to remove or tombstone a deployment, we do not remove the cluster, and return
// the error received from the db. But we should still attempt to mark it's alerts as stale and remove the
// other deployments and their alerts.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorTombstoningDeployments() {
	// We expect alerts to be fetched, and all to be updated.
	alerts := getAlerts(2)
	suite.alerts.EXPECT().ListAlerts(
		testutils.PredMatcher("query for deployment1", func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(alerts, nil)
	for _, alert := range alerts {
		suite.alerts.EXPECT().MarkAlertStale(alert.GetId()).Return(nil)
	}

	suite.alerts.EXPECT().ListAlerts(
		testutils.PredMatcher("query for deployment2", func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment2") })).Return(alerts, nil)
	for _, alert := range alerts {
		suite.alerts.EXPECT().MarkAlertStale(alert.GetId()).Return(nil)
	}

	// Return an error trying to remove the deployments for a cluster.
	deployments := getDeployments(map[string]string{"deployment1": fakeClusterID, "deployment2": fakeClusterID})
	suite.deployments.EXPECT().ListDeployments().Return(deployments, nil)
	expectedErr := fmt.Errorf("issues need tissues")
	suite.deployments.EXPECT().RemoveDeployment("deployment1").Return(expectedErr)
	suite.deployments.EXPECT().RemoveDeployment("deployment2").Return(nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Error(err, "we should receive an error if we can't tombstone one of the deployments")
}

// Test that when no alerts exist for a deployment, everything still functions as intended and the
// deployments and cluster are removed.
func (suite *ClusterDataStoreTestSuite) TestHandlesNoAlerts() {
	// If No alerts exist, everything should still work smoothly.
	suite.alerts.EXPECT().ListAlerts(
		testutils.PredMatcher("query for deployment1", func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(([]*v1.ListAlert)(nil), nil)

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": fakeClusterID, "deployment2": "cluster2"})
	suite.deployments.EXPECT().ListDeployments().Return(deployments, nil)
	suite.deployments.EXPECT().RemoveDeployment("deployment1").Return(nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)
	suite.clusters.EXPECT().RemoveCluster(fakeClusterID).Return(nil)

	suite.nodes.EXPECT().RemoveClusterNodeStore(fakeClusterID).Times(1).Return(nil)

	suite.secrets.EXPECT().SearchListSecrets(search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.GetId()).ProtoQuery()).Return([]*v1.ListSecret{}, nil)

	// run removal.
	suite.clusterDataStore.RemoveCluster(fakeClusterID)
}

// Test that when we fail to get the alerts for a deployment, the deployment and cluster are not removed, and
// the error is returned.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorGettingAlerts() {
	// We expect alerts to be fetched, and all to be updated.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.alerts.EXPECT().ListAlerts(
		testutils.PredMatcher("query for deployment1", func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(([]*v1.ListAlert)(nil), expectedErr)

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": fakeClusterID, "deployment2": "cluster2"})
	suite.deployments.EXPECT().ListDeployments().Return(deployments, nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Error(err, "if we can't fetch the alerts properly, then then deployment and cluster should remain")
}

// Test that when we fail to mark an alert as stale, we do not remove the deployment or the cluster, and
// return the error.
func (suite *ClusterDataStoreTestSuite) TestHandlesErrorUpdatingAlert() {
	// We expect alerts to be fetched, and all to be updated.
	alerts := getAlerts(2)
	suite.alerts.EXPECT().ListAlerts(
		testutils.PredMatcher("query for deployment1", func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(alerts, nil)

	// Let one alert succeed at being updated and one fail.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.alerts.EXPECT().MarkAlertStale(alerts[0].GetId()).Return(expectedErr)
	suite.alerts.EXPECT().MarkAlertStale(alerts[1].GetId()).Return(nil)

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": fakeClusterID, "deployment2": "cluster2"})
	suite.deployments.EXPECT().ListDeployments().Return(deployments, nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &storage.Cluster{
		Id: fakeClusterID,
	}
	suite.clusters.EXPECT().GetCluster(fakeClusterID).Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster(fakeClusterID)
	suite.Error(err, "if we can't mark an alert as stale, then then deployment and cluster should remain")
}

func getAlerts(count int) []*v1.ListAlert {
	alerts := make([]*v1.ListAlert, 0)
	for i := 0; i < count; i++ {
		alert := &v1.ListAlert{
			Id: string(i),
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func getDeployments(deploymentToCluster map[string]string) []*storage.ListDeployment {
	deployments := make([]*storage.ListDeployment, 0)
	for deploymentID, clusterID := range deploymentToCluster {
		deployment := &storage.ListDeployment{
			Id:        deploymentID,
			ClusterId: clusterID,
		}
		deployments = append(deployments, deployment)
	}
	return deployments
}
