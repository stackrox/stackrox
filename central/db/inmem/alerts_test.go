package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestAlerts(t *testing.T) {
	suite.Run(t, new(AlertsTestSuite))
}

type AlertsTestSuite struct {
	suite.Suite
	*InMemoryStore
}

func (suite *AlertsTestSuite) SetupSuite() {
	persistent, err := createBoltDB()
	require.Nil(suite.T(), err)
	suite.InMemoryStore = New(persistent)
}

func (suite *AlertsTestSuite) TeardownSuite() {
	suite.Close()
}

func (suite *AlertsTestSuite) basicAlertTest(updateStore, retrievalStore db.Storage) {
	alerts := []*v1.Alert{
		{
			Id: "id1",
			Policy: &v1.Policy{
				Severity: v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id2",
			Policy: &v1.Policy{
				Severity: v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 100},
		},
	}

	for _, alert := range alerts {
		suite.NoError(updateStore.AddAlert(alert))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, alert := range alerts {
		suite.Error(updateStore.AddAlert(alert))
	}

	// Verify add is persisted
	retrievedAlerts, err := retrievalStore.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal(alerts, retrievedAlerts)

	// Verify update works
	for _, alert := range alerts {
		alert.Policy.Severity = v1.Severity_MEDIUM_SEVERITY
		suite.NoError(updateStore.UpdateAlert(alert))
	}

	retrievedAlerts, err = retrievalStore.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal(alerts, retrievedAlerts)

	// Verify deletion is persisted
	for _, alert := range alerts {
		suite.NoError(updateStore.RemoveAlert(alert.Id))
	}
	retrievedAlerts, err = retrievalStore.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Len(retrievedAlerts, 0)
}

func (suite *AlertsTestSuite) TestPersistence() {
	suite.basicAlertTest(suite.InMemoryStore, suite.persistent)
}

func (suite *AlertsTestSuite) TestAlerts() {
	suite.basicAlertTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *AlertsTestSuite) TestGetAlertsFilters() {
	alert1 := &v1.Alert{
		Id: "id1",
		Policy: &v1.Policy{
			Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
			Name:       "policy1",
			Id:         "policyID1",
			Severity:   v1.Severity_LOW_SEVERITY,
		},
		Deployment: &v1.Deployment{
			ClusterId: "test",
			Id:        "deploymentID1",
			Name:      "deployment1",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Time:  &timestamp.Timestamp{Seconds: 100},
		Stale: true,
	}
	err := suite.AddAlert(alert1)
	suite.NoError(err)
	alert2 := &v1.Alert{
		Id: "id2",
		Policy: &v1.Policy{
			Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
			Name:       "policy2",
			Id:         "policyID2",
			Severity:   v1.Severity_HIGH_SEVERITY,
		},
		Deployment: &v1.Deployment{
			ClusterId: "prod",
			Id:        "deploymentID1",
			Name:      "deployment1",
			Labels: map[string]string{
				"hello": "world",
				"key":   "value",
			},
		},
		Time:  &timestamp.Timestamp{Seconds: 200},
		Stale: false,
	}
	err = suite.AddAlert(alert2)
	suite.NoError(err)

	// Get by ID
	alert, exists, err := suite.GetAlert("id1")
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(alert1, alert)

	cases := []struct {
		name     string
		request  *v1.GetAlertsRequest
		expected []*v1.Alert
	}{
		{
			name:     "all",
			request:  &v1.GetAlertsRequest{},
			expected: []*v1.Alert{alert2, alert1},
		},
		{
			name: "severity",
			request: &v1.GetAlertsRequest{
				Severity: []v1.Severity{
					v1.Severity_HIGH_SEVERITY,
					v1.Severity_LOW_SEVERITY,
				},
			},
			expected: []*v1.Alert{alert2, alert1},
		},
		{
			name: "category",
			request: &v1.GetAlertsRequest{
				Category: []v1.Policy_Category{
					v1.Policy_Category_IMAGE_ASSURANCE,
				},
			},
			expected: []*v1.Alert{alert2, alert1},
		},
		{
			name: "policy name",
			request: &v1.GetAlertsRequest{
				PolicyName: []string{"policy2", "policy23"},
			},
			expected: []*v1.Alert{alert2},
		},
		{
			name: "policy id",
			request: &v1.GetAlertsRequest{
				PolicyId: []string{"policyID1", "randomID"},
			},
			expected: []*v1.Alert{alert1},
		},
		{
			name:     "time since",
			request:  &v1.GetAlertsRequest{Since: &timestamp.Timestamp{Seconds: 150}},
			expected: []*v1.Alert{alert2},
		},
		{
			name:     "deployment id",
			request:  &v1.GetAlertsRequest{DeploymentId: []string{"deploymentID1", "someService"}},
			expected: []*v1.Alert{alert2, alert1},
		},
		{
			name:    "deployment id 2",
			request: &v1.GetAlertsRequest{DeploymentId: []string{"somethingelse"}},
		},
		{
			name:     "deployment name",
			request:  &v1.GetAlertsRequest{DeploymentName: []string{"deployment1"}},
			expected: []*v1.Alert{alert2, alert1},
		},
		{
			name:     "cluster",
			request:  &v1.GetAlertsRequest{Cluster: []string{"test", "someCluster"}},
			expected: []*v1.Alert{alert1},
		},
		{
			name:     "labels",
			request:  &v1.GetAlertsRequest{LabelKey: "key", LabelValue: "value"},
			expected: []*v1.Alert{alert2},
		},
		{
			name:    "labels 2",
			request: &v1.GetAlertsRequest{LabelKey: "key"},
		},
		{
			name: "stale",
			request: &v1.GetAlertsRequest{
				Stale: []bool{false},
			},
			expected: []*v1.Alert{alert2},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			alerts, err := suite.GetAlerts(c.request)
			suite.NoError(err)
			suite.Equal(c.expected, alerts)
		})
	}

	suite.NoError(suite.RemoveAlert(alert1.Id))
	suite.NoError(suite.RemoveAlert(alert2.Id))
}
