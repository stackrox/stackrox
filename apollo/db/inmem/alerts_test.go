package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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
			Id:       "id1",
			Severity: v1.Severity_LOW_SEVERITY,
			Time:     &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id:       "id2",
			Severity: v1.Severity_HIGH_SEVERITY,
			Time:     &timestamp.Timestamp{Seconds: 100},
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
		alert.Severity = v1.Severity_MEDIUM_SEVERITY
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
			Category: v1.Policy_Category_IMAGE_ASSURANCE,
			Name:     "policy1",
		},
		Severity: v1.Severity_LOW_SEVERITY,
		Time:     &timestamp.Timestamp{Seconds: 100},
		Stale:    true,
	}
	err := suite.AddAlert(alert1)
	suite.NoError(err)
	alert2 := &v1.Alert{
		Id: "id2",
		Policy: &v1.Policy{
			Category: v1.Policy_Category_IMAGE_ASSURANCE,
			Name:     "policy2",
		},
		Severity: v1.Severity_HIGH_SEVERITY,
		Time:     &timestamp.Timestamp{Seconds: 200},
		Stale:    false,
	}
	err = suite.AddAlert(alert2)
	suite.NoError(err)

	// Get all alerts
	alerts, err := suite.GetAlerts(&v1.GetAlertsRequest{})
	suite.NoError(err)
	suite.Equal([]*v1.Alert{alert2, alert1}, alerts)

	// Get by ID
	alert, exists, err := suite.GetAlert("id1")
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(alert1, alert)

	// Filter by severity
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{
		Severity: []v1.Severity{
			v1.Severity_HIGH_SEVERITY,
			v1.Severity_LOW_SEVERITY,
		},
	})
	suite.NoError(err)
	suite.Equal([]*v1.Alert{alert2, alert1}, alerts)

	// Filter by category.
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{
		Category: []v1.Policy_Category{
			v1.Policy_Category_IMAGE_ASSURANCE,
		},
	})
	suite.NoError(err)
	suite.Equal([]*v1.Alert{alert2, alert1}, alerts)

	// Filter by Policy.
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{
		PolicyName: []string{"policy2", "policy23"},
	})
	suite.NoError(err)
	suite.Equal([]*v1.Alert{alert2}, alerts)

	// Filter by time.
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{Since: &timestamp.Timestamp{Seconds: 150}})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert2}, alerts)

	// Filter by staleness
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{
		Stale: []bool{false},
	})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert2}, alerts)

	suite.NoError(suite.RemoveAlert(alert1.Id))
	suite.NoError(suite.RemoveAlert(alert2.Id))
}
