package alertmanager

import (
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/processor/mocks"
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	nowProcess        = getProcessIndicator(ptypes.TimestampNow())
	yesterdayProcess  = getProcessIndicator(protoconv.ConvertTimeToTimestamp(time.Now().Add(-24 * time.Hour)))
	twoDaysAgoProcess = getProcessIndicator(protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * 24 * time.Hour)))
)

func getProcessIndicator(timestamp *ptypes.Timestamp) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Signal: &storage.ProcessSignal{
			Name: "apt-get",
			Time: timestamp,
		},
	}
}

func getFakeRuntimeAlert(indicators ...*storage.ProcessIndicator) *storage.Alert {
	v := &storage.Alert_ProcessViolation{Processes: indicators}
	builders.UpdateRuntimeAlertViolationMessage(v)
	return &storage.Alert{
		LifecycleStage:   storage.LifecycleStage_RUNTIME,
		ProcessViolation: v,
	}
}

func TestAlertManager(t *testing.T) {
	suite.Run(t, new(AlertManagerTestSuite))
}

type AlertManagerTestSuite struct {
	suite.Suite

	alertsMock   *alertMocks.MockDataStore
	notifierMock *notifierMocks.MockProcessor

	alertManager AlertManager

	mockCtrl *gomock.Controller
}

func (suite *AlertManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.alertsMock = alertMocks.NewMockDataStore(suite.mockCtrl)
	suite.notifierMock = notifierMocks.NewMockProcessor(suite.mockCtrl)

	suite.alertManager = New(suite.notifierMock, suite.alertsMock, nil)
}

func (suite *AlertManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
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
	suite.alertsMock.EXPECT().SearchRawAlerts(testutils.PredMatcher("query for violation state, policy", queryHasFields(search.ViolationState, search.PolicyID))).Return(([]*storage.Alert)(nil), nil)

	modified, err := suite.alertManager.AlertAndNotify(nil, WithPolicyID("pid"))
	suite.False(modified)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestGetAlertsByDeployment() {
	suite.alertsMock.EXPECT().SearchRawAlerts(testutils.PredMatcher("query for violation state, deployment", queryHasFields(search.ViolationState, search.DeploymentID))).Return(([]*storage.Alert)(nil), nil)

	modified, err := suite.alertManager.AlertAndNotify(nil, WithDeploymentIDs("did"))
	suite.False(modified)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestOnUpdatesWhenAlertsDoNotChange() {
	alerts := getAlerts()

	suite.alertsMock.EXPECT().SearchRawAlerts(gomock.Any()).Return(alerts, nil)
	suite.alertsMock.EXPECT().UpdateAlert(alerts[0]).Return(nil)
	suite.alertsMock.EXPECT().UpdateAlert(alerts[1]).Return(nil)
	suite.alertsMock.EXPECT().UpdateAlert(alerts[2]).Return(nil)

	modified, err := suite.alertManager.AlertAndNotify(alerts)
	suite.True(modified)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMarksOldAlertsStale() {
	alerts := getAlerts()

	suite.alertsMock.EXPECT().MarkAlertStale(alerts[0].GetId()).Return(nil)

	// Next two should be updates with exactly the same values put in.
	suite.alertsMock.EXPECT().UpdateAlert(alerts[1]).Return(nil)
	suite.alertsMock.EXPECT().UpdateAlert(alerts[2]).Return(nil)

	suite.alertsMock.EXPECT().SearchRawAlerts(gomock.Any()).Return(alerts, nil)
	// We should get a notification for the new alert.
	suite.notifierMock.EXPECT().ProcessAlert(alerts[0]).Return()

	// Make one of the alerts not appear in the current alerts.
	modified, err := suite.alertManager.AlertAndNotify(alerts[1:])
	suite.True(modified)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestSendsNotificationsForNewAlerts() {
	alerts := getAlerts()

	// PolicyUpsert side effects. We won't have any deployments or alerts yet.
	suite.alertsMock.EXPECT().UpdateAlert(alerts[0]).Return(nil)
	suite.alertsMock.EXPECT().UpdateAlert(alerts[1]).Return(nil)
	suite.alertsMock.EXPECT().UpdateAlert(alerts[2]).Return(nil)

	// We should get a notification for the new alert.
	suite.notifierMock.EXPECT().ProcessAlert(alerts[0]).Return()

	// Make one of the alerts not appear in the previous alerts.
	suite.alertsMock.EXPECT().SearchRawAlerts(gomock.Any()).Return(alerts[1:], nil)

	modified, err := suite.alertManager.AlertAndNotify(alerts)
	suite.True(modified)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) makeAlertsMockReturn(alerts ...*storage.Alert) {
	suite.alertsMock.EXPECT().SearchRawAlerts(
		testutils.PredMatcher("query for violation state, deployment, policy", queryHasFields(search.ViolationState, search.DeploymentID, search.PolicyID))).
		Return(alerts, nil)
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesForNonRuntime() {
	suite.False(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(getAlerts()[0]))
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesWithNoOldAlert() {
	suite.makeAlertsMockReturn()
	alert := getFakeRuntimeAlert(nowProcess)
	clonedAlert := protoutils.CloneStorageAlert(alert)
	suite.False(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(alert))
	suite.Equal(clonedAlert, alert)
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesWithTheSameAlert() {
	suite.makeAlertsMockReturn(getFakeRuntimeAlert(nowProcess))
	suite.True(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(getFakeRuntimeAlert(nowProcess)))
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesWithAnOldAlert() {
	suite.makeAlertsMockReturn(getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess))
	alert := getFakeRuntimeAlert(nowProcess)
	clonedAlert := protoutils.CloneStorageAlert(alert)
	suite.False(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(alert))
	suite.Equal(clonedAlert, alert)
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesWithOldAndResolved() {
	suite.makeAlertsMockReturn(getFakeRuntimeAlert(nowProcess), getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess))
	suite.True(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(getFakeRuntimeAlert(nowProcess)))
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesWithSuperOldAlert() {
	suite.makeAlertsMockReturn(getFakeRuntimeAlert(nowProcess), getFakeRuntimeAlert(twoDaysAgoProcess, nowProcess))
	suite.True(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(getFakeRuntimeAlert(yesterdayProcess)))
}

func (suite *AlertManagerTestSuite) TestTrimResolvedProcessesActuallyTrims() {
	suite.makeAlertsMockReturn(getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess))
	alert := getFakeRuntimeAlert(yesterdayProcess, nowProcess)
	clonedAlert := protoutils.CloneStorageAlert(alert)
	suite.False(suite.alertManager.(*alertManagerImpl).trimResolvedProcessesFromRuntimeAlert(alert))
	suite.NotEqual(clonedAlert, alert)
	suite.Len(alert.GetProcessViolation().GetProcesses(), 1)
	suite.Equal(alert.GetProcessViolation().GetProcesses()[0], nowProcess)
}

func TestMergeProcessesFromOldIntoNew(t *testing.T) {
	for _, c := range []struct {
		desc           string
		old            *storage.Alert
		new            *storage.Alert
		expectedNew    *storage.Alert
		expectedOutput bool
	}{
		{
			desc:           "Equal",
			old:            getFakeRuntimeAlert(yesterdayProcess),
			new:            getFakeRuntimeAlert(yesterdayProcess),
			expectedNew:    nil,
			expectedOutput: false,
		},
		{
			desc:           "Equal with two",
			old:            getFakeRuntimeAlert(yesterdayProcess, nowProcess),
			new:            getFakeRuntimeAlert(yesterdayProcess, nowProcess),
			expectedOutput: false,
		},
		{
			desc:           "New has new",
			old:            getFakeRuntimeAlert(yesterdayProcess),
			new:            getFakeRuntimeAlert(nowProcess),
			expectedNew:    getFakeRuntimeAlert(yesterdayProcess, nowProcess),
			expectedOutput: true,
		},
		{
			desc:           "New has many new",
			old:            getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess),
			new:            getFakeRuntimeAlert(yesterdayProcess, nowProcess),
			expectedNew:    getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess, nowProcess),
			expectedOutput: true,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			out := mergeProcessesFromOldIntoNew(c.old, c.new)
			assert.Equal(t, c.expectedOutput, out)
			if c.expectedNew != nil {
				assert.Equal(t, c.expectedNew, c.new)
			}
		})
	}
}

//////////////////////////////////////
// TEST DATA
///////////////////////////////////////

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getAlerts() []*storage.Alert {
	return []*storage.Alert{
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
func getDeployments() []*storage.Deployment {
	return []*storage.Deployment{
		{
			Name: "deployment1",
			Containers: []*storage.Container{
				{
					Image: &storage.Image{
						Name: &storage.ImageName{
							Tag:    "latest1",
							Remote: "stackrox/health",
						},
					},
				},
			},
		},
		{
			Name: "deployment2",
			Containers: []*storage.Container{
				{
					Image: &storage.Image{
						Name: &storage.ImageName{
							Tag:    "latest2",
							Remote: "stackrox/health",
						},
					},
				},
			},
		},
		{
			Name: "deployment3",
			Containers: []*storage.Container{
				{
					Image: &storage.Image{
						Name: &storage.ImageName{
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
func getPolicies() []*storage.Policy {
	return []*storage.Policy{
		{
			Id:         "policy1",
			Name:       "latest1",
			Severity:   storage.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			Fields: &storage.PolicyFields{
				ImageName: &storage.ImageNamePolicy{
					Tag: "latest1",
				},
			},
		},
		{
			Id:         "policy2",
			Name:       "latest2",
			Severity:   storage.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			Fields: &storage.PolicyFields{
				ImageName: &storage.ImageNamePolicy{
					Tag: "latest2",
				},
			},
		},
		{
			Id:         "policy3",
			Name:       "latest3",
			Severity:   storage.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			Fields: &storage.PolicyFields{
				ImageName: &storage.ImageNamePolicy{
					Tag: "latest3",
				},
			},
		},
	}
}
