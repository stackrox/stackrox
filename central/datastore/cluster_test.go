package datastore

import (
	"fmt"
	"testing"

	"strings"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestPolicyValidator(t *testing.T) {
	suite.Run(t, new(PolicyValidatorTestSuite))
}

type PolicyValidatorTestSuite struct {
	suite.Suite

	clusterStorage      *db.MockClusterStorage
	deploymentDataStore *MockDeploymentDataStore
	alertDataStore      *MockAlertDataStore

	clusterDataStore ClusterDataStore
}

func (suite *PolicyValidatorTestSuite) SetupTest() {
	suite.clusterStorage = &db.MockClusterStorage{}
	suite.deploymentDataStore = &MockDeploymentDataStore{}
	suite.alertDataStore = &MockAlertDataStore{}

	suite.clusterDataStore = NewClusterDataStore(suite.clusterStorage, suite.deploymentDataStore, suite.alertDataStore)
}

// Test the happy path.
func (suite *PolicyValidatorTestSuite) TestRemoveTombstonesDeploymentsAndMarksAlertsStale() {
	// We expect alerts to be fetched, and all to be updated.
	alerts := getAlerts(2)
	suite.alertDataStore.On("GetAlerts",
		mock.MatchedBy(func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(alerts, nil)
	for _, alert := range alerts {
		suite.alertDataStore.On("UpdateAlert", alert).Return(nil)
	}

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": "cluster1", "deployment2": "cluster2"})
	suite.deploymentDataStore.On("GetDeployments").Return(deployments, nil)
	suite.deploymentDataStore.On("RemoveDeployment", "deployment1").Return(nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)
	suite.clusterStorage.On("RemoveCluster", "cluster1").Return(nil)

	// run removal.
	suite.clusterDataStore.RemoveCluster("cluster1")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.alertDataStore.AssertExpectations(suite.T())
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when the cluster we try to remove does not exist, we return an error.
func (suite *PolicyValidatorTestSuite) TestHandlesClusterDoesNotExist() {
	// Return false for the cluster not existing.
	suite.clusterStorage.On("GetCluster", "cluster1").Return((*v1.Cluster)(nil), false, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster("cluster1")
	suite.Error(err, "expected an error since the cluster did not exist")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when we cannot fetch a cluster, we return the error from the DB.
func (suite *PolicyValidatorTestSuite) TestHandlesErrorGettingCluster() {
	// Return an error trying to fetch the cluster.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.clusterStorage.On("GetCluster", "cluster1").Return((*v1.Cluster)(nil), true, expectedErr)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster("cluster1")
	suite.Equal(expectedErr, err)

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when no deployments exist for a cluster, the cluster is removed successfully with no additional
// operations on either deployments or alerts.
func (suite *PolicyValidatorTestSuite) TestHandlesNoDeployments() {
	// Return an error trying to fetch the deployments for a cluster.
	suite.deploymentDataStore.On("GetDeployments").Return(([]*v1.Deployment)(nil), nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)
	suite.clusterStorage.On("RemoveCluster", "cluster1").Return(nil)

	// run removal.
	suite.clusterDataStore.RemoveCluster("cluster1")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when we get an error trying to fetch the deployments for a cluster, we do not remove the cluster
// and instead return the error.
func (suite *PolicyValidatorTestSuite) TestHandlesErrorGettingDeployments() {
	// Return an error trying to fetch the deployments for a cluster.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.deploymentDataStore.On("GetDeployments").Return(([]*v1.Deployment)(nil), expectedErr)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster("cluster1")
	suite.Equal(expectedErr, err)

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when we are unable to remove or tombstone a deployment, we do not remove the cluster, and return
// the error received from the db. But we should still attempt to mark it's alerts as stale and remove the
// other deployments and their alerts.
func (suite *PolicyValidatorTestSuite) TestHandlesErrorTombstoningDeployments() {
	// We expect alerts to be fetched, and all to be updated.
	alerts := getAlerts(2)
	suite.alertDataStore.On("GetAlerts",
		mock.MatchedBy(func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(alerts, nil)
	for _, alert := range alerts {
		suite.alertDataStore.On("UpdateAlert", alert).Return(nil)
	}

	suite.alertDataStore.On("GetAlerts",
		mock.MatchedBy(func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment2") })).Return(alerts, nil)
	for _, alert := range alerts {
		suite.alertDataStore.On("UpdateAlert", alert).Return(nil)
	}

	// Return an error trying to remove the deployments for a cluster.
	deployments := getDeployments(map[string]string{"deployment1": "cluster1", "deployment2": "cluster1"})
	suite.deploymentDataStore.On("GetDeployments").Return(deployments, nil)
	expectedErr := fmt.Errorf("issues need tissues")
	suite.deploymentDataStore.On("RemoveDeployment", "deployment1").Return(expectedErr)
	suite.deploymentDataStore.On("RemoveDeployment", "deployment2").Return(nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster("cluster1")
	suite.Error(err, "we should receive an error if we can't tombstone one of the deployments")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.alertDataStore.AssertExpectations(suite.T())
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when no alerts exist for a deployment, everything still functions as intended and the
// deployments and cluster are removed.
func (suite *PolicyValidatorTestSuite) TestHandlesNoAlerts() {
	// If No alerts exist, everything should still work smoothly.
	suite.alertDataStore.On("GetAlerts",
		mock.MatchedBy(func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(([]*v1.Alert)(nil), nil)

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": "cluster1", "deployment2": "cluster2"})
	suite.deploymentDataStore.On("GetDeployments").Return(deployments, nil)
	suite.deploymentDataStore.On("RemoveDeployment", "deployment1").Return(nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)
	suite.clusterStorage.On("RemoveCluster", "cluster1").Return(nil)

	// run removal.
	suite.clusterDataStore.RemoveCluster("cluster1")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.alertDataStore.AssertExpectations(suite.T())
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when we fail to get the alerts for a deployment, the deployment and cluster are not removed, and
// the error is returned.
func (suite *PolicyValidatorTestSuite) TestHandlesErrorGettingAlerts() {
	// We expect alerts to be fetched, and all to be updated.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.alertDataStore.On("GetAlerts",
		mock.MatchedBy(func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(([]*v1.Alert)(nil), expectedErr)

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": "cluster1", "deployment2": "cluster2"})
	suite.deploymentDataStore.On("GetDeployments").Return(deployments, nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster("cluster1")
	suite.Error(err, "if we can't fetch the alerts properly, then then deployment and cluster should remain")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.alertDataStore.AssertExpectations(suite.T())
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

// Test that when we fail to mark an alert as stale, we do not remove the deployment or the cluster, and
// return the error.
func (suite *PolicyValidatorTestSuite) TestHandlesErrorUpdatingAlert() {
	// We expect alerts to be fetched, and all to be updated.
	alerts := getAlerts(2)
	suite.alertDataStore.On("GetAlerts",
		mock.MatchedBy(func(req *v1.ListAlertsRequest) bool { return strings.Contains(req.Query, "deployment1") })).Return(alerts, nil)

	// Let one alert succeed at being updated and one fail.
	expectedErr := fmt.Errorf("issues need tissues")
	suite.alertDataStore.On("UpdateAlert", alerts[0]).Return(expectedErr)
	suite.alertDataStore.On("UpdateAlert", alerts[1]).Return(nil)

	// We expect deployments to be fetched, and only those for cluster1 to be tombstoned.
	deployments := getDeployments(map[string]string{"deployment1": "cluster1", "deployment2": "cluster2"})
	suite.deploymentDataStore.On("GetDeployments").Return(deployments, nil)

	// Return a cluster with an id that matches the deployments we want to tombstone.
	cluster := &v1.Cluster{
		Id: "cluster1",
	}
	suite.clusterStorage.On("GetCluster", "cluster1").Return(cluster, true, nil)

	// run removal.
	err := suite.clusterDataStore.RemoveCluster("cluster1")
	suite.Error(err, "if we can't mark an alert as stale, then then deployment and cluster should remain")

	// Make sure the proper storage interactions happened with deployments and alerts.
	suite.alertDataStore.AssertExpectations(suite.T())
	suite.deploymentDataStore.AssertExpectations(suite.T())
	suite.clusterStorage.AssertExpectations(suite.T())
}

func getAlerts(count int) []*v1.Alert {
	alerts := make([]*v1.Alert, 0)
	for i := 0; i < count; i++ {
		alert := &v1.Alert{
			Id: string(i),
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func getDeployments(deploymentToCluster map[string]string) []*v1.Deployment {
	deployments := make([]*v1.Deployment, 0)
	for deploymentID, clusterID := range deploymentToCluster {
		deployment := &v1.Deployment{
			Id:        deploymentID,
			ClusterId: clusterID,
		}
		deployments = append(deployments, deployment)
	}
	return deployments
}
