package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/assert"
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
	alert1 := &v1.Alert{
		Id:       "id1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := updateStore.AddAlert(alert1)
	suite.Nil(err)
	alert2 := &v1.Alert{
		Id:       "id2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	err = updateStore.AddAlert(alert2)
	suite.Nil(err)

	// Verify add is persisted
	alerts, err := retrievalStore.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	// Verify update works
	alert1.Severity = v1.Severity_HIGH_SEVERITY
	err = updateStore.UpdateAlert(alert1)
	suite.Nil(err)
	alerts, err = retrievalStore.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.Alert{alert1, alert2}, alerts)

	// Verify deletion is persisted
	err = updateStore.RemoveAlert(alert1.Id)
	suite.Nil(err)
	err = updateStore.RemoveAlert(alert2.Id)
	suite.Nil(err)
	alerts, err = retrievalStore.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	suite.Len(alerts, 0)
}

func (suite *AlertsTestSuite) TestPersistence() {
	suite.basicAlertTest(suite.InMemoryStore, suite.persistent)
}

func (suite *AlertsTestSuite) TestAlerts() {
	suite.basicAlertTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *AlertsTestSuite) TestGetAlertsFilters() {
	alert1 := &v1.Alert{
		Id:       "id1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := suite.AddAlert(alert1)
	suite.Nil(err)
	alert2 := &v1.Alert{
		Id:       "id2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	err = suite.AddAlert(alert2)
	suite.Nil(err)

	// Get all alerts
	alerts, err := suite.GetAlerts(&v1.GetAlertsRequest{})
	suite.Nil(err)
	assert.Equal(suite.T(), []*v1.Alert{alert1, alert2}, alerts)

	// Get by ID
	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{Id: "id1"})
	suite.Nil(err)
	assert.Equal(suite.T(), []*v1.Alert{alert1}, alerts)

	alerts, err = suite.GetAlerts(&v1.GetAlertsRequest{Severity: v1.Severity_HIGH_SEVERITY})
	suite.Nil(err)
	assert.Equal(suite.T(), []*v1.Alert{alert2}, alerts)
}
