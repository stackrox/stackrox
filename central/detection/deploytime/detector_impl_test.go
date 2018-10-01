package deploytime

import (
	"reflect"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	deploymentDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/detection/deployment"
	utilsMocks "github.com/stackrox/rox/central/detection/utils/mocks"
	enrichmentMocks "github.com/stackrox/rox/central/enrichment/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestDetector(t *testing.T) {
	t.Skip("TODO(viswa): This test needs to be refactored to make sense in a search-based policies world.")
	suite.Run(t, new(DetectorTestSuite))
}

type DetectorTestSuite struct {
	suite.Suite

	alertManagerMock *utilsMocks.AlertManager
	enricherMock     *enrichmentMocks.Enricher
	deploymentsMock  *deploymentDataStoreMocks.DataStore

	detector Detector
}

func (suite *DetectorTestSuite) SetupTest() {
	suite.alertManagerMock = &utilsMocks.AlertManager{}
	suite.enricherMock = &enrichmentMocks.Enricher{}
	suite.deploymentsMock = &deploymentDataStoreMocks.DataStore{}

	suite.detector = NewDetector(
		deployment.NewPolicySet(&policyMocks.DataStore{}),
		suite.alertManagerMock,
		suite.enricherMock,
		suite.deploymentsMock)
}

// Happy path for adding a deployment that fits a policy.
func (suite *DetectorTestSuite) TestDeploymentUpdatedFitsPolicy() {
	deployments := getDeployments()
	policies := getPolicies()
	alerts := getAlerts()

	// PolicyUpsert side effects. We won't have any deployments or alerts yet.
	suite.enricherMock.On("ReprocessRiskAsync").Return(nil)
	suite.deploymentsMock.On("GetDeployments").Return(([]*v1.Deployment)(nil), nil)
	suite.alertManagerMock.On("GetAlertsByPolicy", policies[0].GetId()).Return(([]*v1.Alert)(nil), nil)
	suite.alertManagerMock.On("AlertAndNotify", ([]*v1.Alert)(nil), ([]*v1.Alert)(nil)).Return(nil)

	// DeploymentUpdate side effects. We won't have any alerts yet, but will generate one.
	suite.enricherMock.On("Enrich", deployments[0]).Return(true, nil)
	suite.enricherMock.On("ReprocessDeploymentRiskAsync", deployments[0]).Return(nil)
	suite.alertManagerMock.On("GetAlertsByDeployment", deployments[0].GetId()).Return(([]*v1.Alert)(nil), nil)
	suite.alertManagerMock.On("AlertAndNotify", ([]*v1.Alert)(nil), mock.MatchedBy(func(in interface{}) bool {
		alertList := in.([]*v1.Alert)
		return len(alertList) == 1 && AlertsEqual(alerts[0], alertList[0])
	})).Return(nil)

	// Add policy then process a deployment that violates it.
	err := suite.detector.UpsertPolicy(policies[0])
	suite.NoError(err, "upsert policy should succeed")
	_, _, err = suite.detector.DeploymentUpdated(deployments[0])
	suite.NoError(err, "deployment update should succeed")

	// Assert mocks have been called as expected.
	suite.alertManagerMock.AssertExpectations(suite.T())
	suite.enricherMock.AssertExpectations(suite.T())
	suite.deploymentsMock.AssertExpectations(suite.T())
}

// Test the happy path for removing deployments and having all of their alerts marked stale.
func (suite *DetectorTestSuite) TestRemovedDeploymentsAlertsMarkedStale() {
	deployments := getDeployments()
	alerts := getAlerts()

	// DeploymentUpdate side effects. We won't have any alerts yet, but will generate one.
	suite.alertManagerMock.On("GetAlertsByDeployment", deployments[0].GetId()).Return(alerts, nil)
	suite.alertManagerMock.On("AlertAndNotify", alerts, ([]*v1.Alert)(nil)).Return(nil)

	// Remove deployment.
	err := suite.detector.DeploymentRemoved(deployments[0])
	suite.NoError(err, "deployment update should succeed")

	// Assert mocks have been called as expected.
	suite.alertManagerMock.AssertExpectations(suite.T())
	suite.enricherMock.AssertExpectations(suite.T())
	suite.deploymentsMock.AssertExpectations(suite.T())
}

// Happy path for adding a policy that fits a deployment.
func (suite *DetectorTestSuite) TestPolicyUpsertedFitsDeployment() {
	deployments := getDeployments()
	policies := getPolicies()
	alerts := getAlerts()

	// PolicyUpsert side effects.
	suite.enricherMock.On("ReprocessRiskAsync").Return(nil)
	suite.deploymentsMock.On("GetDeployments").Return(deployments, nil)
	suite.alertManagerMock.On("GetAlertsByPolicy", policies[0].GetId()).Return(([]*v1.Alert)(nil), nil)
	suite.alertManagerMock.On("AlertAndNotify", ([]*v1.Alert)(nil), mock.MatchedBy(func(in interface{}) bool {
		alertList := in.([]*v1.Alert)
		return len(alertList) == 1 && AlertsEqual(alerts[0], alertList[0])
	})).Return(nil)

	// Add policy then process a deployment that violates it.
	err := suite.detector.UpsertPolicy(policies[0])
	suite.NoError(err, "upsert policy should succeed")

	// Assert mocks have been called as expected.
	suite.alertManagerMock.AssertExpectations(suite.T())
	suite.enricherMock.AssertExpectations(suite.T())
	suite.deploymentsMock.AssertExpectations(suite.T())
}

// Test the happy path for removing policies and having all of their alerts marked stale.
func (suite *DetectorTestSuite) TestRemovedPoliciesAlertsMarkedStale() {
	policies := getPolicies()
	alerts := getAlerts()

	// DeploymentUpdate side effects. We won't have any alerts yet, but will generate one.
	suite.alertManagerMock.On("GetAlertsByPolicy", policies[0].GetId()).Return(alerts, nil)
	suite.alertManagerMock.On("AlertAndNotify", alerts, ([]*v1.Alert)(nil)).Return(nil)

	// Remove deployment.
	err := suite.detector.RemovePolicy(policies[0].GetId())
	suite.NoError(err, "deployment update should succeed")

	// Assert mocks have been called as expected.
	suite.alertManagerMock.AssertExpectations(suite.T())
	suite.enricherMock.AssertExpectations(suite.T())
	suite.deploymentsMock.AssertExpectations(suite.T())
}

// Helper Functions
////////////////////

// AlertsEqual checks to make sure the deployment, policy, and enforcement information of two alerts is the same.
func AlertsEqual(a1, a2 *v1.Alert) bool {
	return reflect.DeepEqual(a1.Policy, a2.Policy) &&
		reflect.DeepEqual(a1.Deployment, a2.Deployment) &&
		reflect.DeepEqual(a1.Enforcement, a2.Enforcement)
}

// Test Data
/////////////

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getAlerts() []*v1.Alert {
	return []*v1.Alert{
		{
			Id:         "alert1",
			Policy:     getPolicies()[0],
			Deployment: getDeployments()[0],
			Time:       &ptypes.Timestamp{Seconds: 100},
		},
		{
			Id:         "alert2",
			Policy:     getPolicies()[1],
			Deployment: getDeployments()[1],
			Time:       &ptypes.Timestamp{Seconds: 200},
		},
		{
			Id:         "alert3",
			Policy:     getPolicies()[2],
			Deployment: getDeployments()[2],
			Time:       &ptypes.Timestamp{Seconds: 300},
		},
	}
}

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getDeployments() []*v1.Deployment {
	return []*v1.Deployment{
		{
			Name: "deployment1",
			Containers: []*v1.Container{
				{
					Image: &v1.Image{
						Name: &v1.ImageName{
							Tag:    "latest1",
							Remote: "stackrox/health",
						},
					},
				},
			},
		},
		{
			Name: "deployment2",
			Containers: []*v1.Container{
				{
					Image: &v1.Image{
						Name: &v1.ImageName{
							Tag:    "latest2",
							Remote: "stackrox/health",
						},
					},
				},
			},
		},
		{
			Name: "deployment3",
			Containers: []*v1.Container{
				{
					Image: &v1.Image{
						Name: &v1.ImageName{
							Tag:    "latest3",
							Remote: "stackrox/health",
						},
					},
				},
			},
		},
	}
}

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getPolicies() []*v1.Policy {
	return []*v1.Policy{
		{
			Id:         "policy1",
			Name:       "latest1",
			Severity:   v1.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			Fields: &v1.PolicyFields{
				ImageName: &v1.ImageNamePolicy{
					Tag: "latest1",
				},
			},
		},
		{
			Id:         "policy2",
			Name:       "latest2",
			Severity:   v1.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			Fields: &v1.PolicyFields{
				ImageName: &v1.ImageNamePolicy{
					Tag: "latest2",
				},
			},
		},
		{
			Id:         "policy3",
			Name:       "latest3",
			Severity:   v1.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			Fields: &v1.PolicyFields{
				ImageName: &v1.ImageNamePolicy{
					Tag: "latest3",
				},
			},
		},
	}
}
