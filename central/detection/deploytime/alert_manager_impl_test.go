package deploytime

import (
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/processor/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestAlertManager(t *testing.T) {
	suite.Run(t, new(AlertManagerTestSuite))
}

type AlertManagerTestSuite struct {
	suite.Suite

	alertsMock   *alertMocks.DataStore
	notifierMock *notifierMocks.Processor

	alertManager AlertManager
}

func (suite *AlertManagerTestSuite) SetupTest() {
	suite.alertsMock = &alertMocks.DataStore{}
	suite.notifierMock = &notifierMocks.Processor{}

	suite.alertManager = NewAlertManager(suite.notifierMock, suite.alertsMock)
}

// Returns a function that can be used to match *v1.Query,
// which ensure that the query specifies all the fields.
func queryHasFields(fields ...search.FieldLabel) func(interface{}) bool {
	return func(in interface{}) bool {
		q := in.(*v1.Query)

		fieldsFound := make([]bool, len(fields))
		search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
			mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
			if !ok {
				return
			}
			for i, field := range fields {
				if mfQ.MatchFieldQuery.GetField() == field.String() {
					fieldsFound[i] = true
				}
			}
		})

		for _, found := range fieldsFound {
			if !found {
				return false
			}
		}
		return true
	}
}

func (suite *AlertManagerTestSuite) TestGetAlertsByPolicy() {
	// PolicyUpsert side effects. We won't have any deployments or alerts yet.
	suite.alertsMock.On("SearchRawAlerts", mock.MatchedBy(queryHasFields(search.Stale, search.PolicyID))).Return(([]*v1.Alert)(nil), nil)

	_, err := suite.alertManager.GetAlertsByPolicy("pid")
	suite.NoError(err, "update should succeed")

	suite.alertsMock.AssertExpectations(suite.T())
	suite.notifierMock.AssertExpectations(suite.T())
}

func (suite *AlertManagerTestSuite) TestGetAlertsByDeployment() {
	// PolicyUpsert side effects. We won't have any deployments or alerts yet.
	suite.alertsMock.On("SearchRawAlerts", mock.MatchedBy(queryHasFields(search.Stale, search.DeploymentID))).Return(([]*v1.Alert)(nil), nil)

	_, err := suite.alertManager.GetAlertsByDeployment("did")
	suite.NoError(err, "update should succeed")

	suite.alertsMock.AssertExpectations(suite.T())
	suite.notifierMock.AssertExpectations(suite.T())
}

func (suite *AlertManagerTestSuite) TestOnUpdatesWhenAlertsDoNotChange() {
	alerts := getAlerts()

	// PolicyUpsert side effects. We won't have any deployments or alerts yet.
	suite.alertsMock.On("UpdateAlert", alerts[0]).Return(nil)
	suite.alertsMock.On("UpdateAlert", alerts[1]).Return(nil)
	suite.alertsMock.On("UpdateAlert", alerts[2]).Return(nil)

	err := suite.alertManager.AlertAndNotify(alerts, alerts)
	suite.NoError(err, "update should succeed")

	suite.alertsMock.AssertExpectations(suite.T())
	suite.notifierMock.AssertExpectations(suite.T())
}

func (suite *AlertManagerTestSuite) TestMarksOldAlertsStale() {
	alerts := getAlerts()

	// First alert match on being stale.
	suite.alertsMock.On("UpdateAlert", mock.MatchedBy(func(in interface{}) bool {
		alert := in.(*v1.Alert)
		return alert.Stale
	})).Return(nil)

	// Next two should be updates with exactly the same values put in.
	suite.alertsMock.On("UpdateAlert", alerts[1]).Return(nil)
	suite.alertsMock.On("UpdateAlert", alerts[2]).Return(nil)

	// Make one of the alerts not appear in the current alerts.
	err := suite.alertManager.AlertAndNotify(alerts, alerts[1:])
	suite.NoError(err, "update should succeed")

	suite.alertsMock.AssertExpectations(suite.T())
	suite.notifierMock.AssertExpectations(suite.T())
}

func (suite *AlertManagerTestSuite) TestSendsNotificationsForNewAlerts() {
	alerts := getAlerts()

	// PolicyUpsert side effects. We won't have any deployments or alerts yet.
	suite.alertsMock.On("UpdateAlert", alerts[0]).Return(nil)
	suite.alertsMock.On("UpdateAlert", alerts[1]).Return(nil)
	suite.alertsMock.On("UpdateAlert", alerts[2]).Return(nil)

	// We should get a notification for the new alert.
	suite.notifierMock.On("ProcessAlert", alerts[0]).Return(nil)

	// Make one of the alerts not appear in the previous alerts.
	err := suite.alertManager.AlertAndNotify(alerts[1:], alerts)
	suite.NoError(err, "update should succeed")

	suite.alertsMock.AssertExpectations(suite.T())
	suite.notifierMock.AssertExpectations(suite.T())
}

//////////////////////////////////////
// TEST DATA
///////////////////////////////////////

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
