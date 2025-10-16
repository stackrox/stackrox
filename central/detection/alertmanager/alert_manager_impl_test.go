package alertmanager

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	"github.com/stackrox/rox/central/detection"
	runtimeDetectorMocks "github.com/stackrox/rox/central/detection/runtime/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

var (
	now        = time.Now()
	yesterday  = now.Add(-24 * time.Hour)
	twoDaysAgo = now.Add(-2 * 24 * time.Hour)

	nowProcess        = getProcessIndicator(now)
	yesterdayProcess  = getProcessIndicator(yesterday)
	twoDaysAgoProcess = getProcessIndicator(twoDaysAgo)

	firstKubeEventViolation  = getKubeEventViolation("1", yesterday)
	secondKubeEventViolation = getKubeEventViolation("2", now)

	firstNetworkFlowViolation  = getNetworkFlowViolation("1", yesterday)
	secondNetworkFlowViolation = getNetworkFlowViolation("2", now)
)

func getKubeEventViolation(msg string, violationTime time.Time) *storage.Alert_Violation {
	av := &storage.Alert_Violation{}
	av.SetMessage(msg)
	av.SetType(storage.Alert_Violation_K8S_EVENT)
	av.SetTime(protocompat.ConvertTimeToTimestampOrNil(&violationTime))
	return av
}

func getNetworkFlowViolation(msg string, networkFlowTimestamp time.Time) *storage.Alert_Violation {
	return storage.Alert_Violation_builder{
		Message: msg,
		KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
			Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
				storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{
					Key:   "NetworkFlowTimestamp",
					Value: networkFlowTimestamp.Format("2006-01-02 15:04:05 UTC"),
				}.Build(),
			},
		}.Build(),
		Type: storage.Alert_Violation_NETWORK_FLOW,
	}.Build()
}

func getProcessIndicator(processTime time.Time) *storage.ProcessIndicator {
	ps := &storage.ProcessSignal{}
	ps.SetName("apt-get")
	ps.SetTime(protocompat.ConvertTimeToTimestampOrNil(&processTime))
	pi := &storage.ProcessIndicator{}
	pi.SetSignal(ps)
	return pi
}

func getFakeRuntimeAlert(indicators ...*storage.ProcessIndicator) *storage.Alert {
	v := &storage.Alert_ProcessViolation{}
	v.SetProcesses(indicators)
	printer.UpdateProcessAlertViolationMessage(v)
	alert := &storage.Alert{}
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	alert.SetProcessViolation(v)
	return alert
}

func getFakeResourceRuntimeAlert(resourceType storage.Alert_Resource_ResourceType, resourceName, clusterID, namespaceID, namespace string) *storage.Alert {
	ar := &storage.Alert_Resource{}
	ar.SetResourceType(resourceType)
	ar.SetName(resourceName)
	ar.SetClusterId(clusterID)
	ar.SetClusterName("prod cluster")
	ar.SetNamespace(namespace)
	ar.SetNamespaceId(namespaceID)
	alert := &storage.Alert{}
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	alert.SetResource(proto.ValueOrDefault(ar))
	return alert
}

func appendViolations(alert *storage.Alert, violations ...*storage.Alert_Violation) *storage.Alert {
	alert.SetViolations(append(alert.GetViolations(), violations...))
	return alert
}

func TestAlertManager(t *testing.T) {
	suite.Run(t, new(AlertManagerTestSuite))
}

type AlertManagerTestSuite struct {
	suite.Suite

	alertsMock          *alertMocks.MockDataStore
	notifierMock        *notifierMocks.MockProcessor
	runtimeDetectorMock *runtimeDetectorMocks.MockDetector
	policySet           detection.PolicySet

	alertManager AlertManager

	mockCtrl *gomock.Controller
	ctx      context.Context
}

func (suite *AlertManagerTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.alertsMock = alertMocks.NewMockDataStore(suite.mockCtrl)
	suite.notifierMock = notifierMocks.NewMockProcessor(suite.mockCtrl)
	suite.runtimeDetectorMock = runtimeDetectorMocks.NewMockDetector(suite.mockCtrl)
	suite.policySet = detection.NewPolicySet(policyMocks.NewMockDataStore(suite.mockCtrl))

	suite.alertManager = New(suite.notifierMock, suite.alertsMock, suite.runtimeDetectorMock)
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

func (suite *AlertManagerTestSuite) TestNotifyAndUpdateBatch() {
	alerts := []*storage.Alert{fixtures.GetAlert(), fixtures.GetAlert()}
	alerts[0].GetPolicy().SetId("Pol1")
	alerts[0].GetDeployment().SetId("Dep1")
	alerts[1].GetPolicy().SetId("Pol2")
	alerts[1].GetDeployment().SetId("Dep2")

	suite.T().Setenv(env.AlertRenotifDebounceDuration.EnvVar(), "5m")

	resolvedAlerts := []*storage.Alert{alerts[0].CloneVT(), alerts[1].CloneVT()}
	resolvedAlerts[0].SetResolvedAt(protoconv.MustConvertTimeToTimestamp(time.Now().Add(-10 * time.Minute)))
	resolvedAlerts[1].SetResolvedAt(protoconv.MustConvertTimeToTimestamp(time.Now().Add(-2 * time.Minute)))

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx,
		testutils.PredMatcher("query for dep 1", func(q *v1.Query) bool {
			return strings.Contains(protocompat.MarshalTextString(q), "Dep1")
		}), false).Return([]*storage.Alert{resolvedAlerts[0]}, nil)
	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx,
		testutils.PredMatcher("query for dep 2", func(q *v1.Query) bool {
			return strings.Contains(protocompat.MarshalTextString(q), "Dep2")
		}), false).Return([]*storage.Alert{resolvedAlerts[1]}, nil)

	// Only the first alert will get notified
	suite.notifierMock.EXPECT().ProcessAlert(suite.ctx, alerts[0])
	// All alerts will still get inserted
	for _, alert := range alerts {
		suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, alert)
	}
	suite.NoError(suite.alertManager.(*alertManagerImpl).notifyAndUpdateBatch(suite.ctx, alerts))
}

func (suite *AlertManagerTestSuite) TestGetAlertsByPolicy() {
	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, testutils.PredMatcher("query for violation state, policy", queryHasFields(search.ViolationState, search.PolicyID)), true).Return(([]*storage.Alert)(nil), nil)

	modified, err := suite.alertManager.AlertAndNotify(suite.ctx, nil, WithPolicyID("pid"))
	suite.False(modified.Cardinality() > 0)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestGetAlertsByDeployment() {
	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, testutils.PredMatcher("query for violation state, deployment", queryHasFields(search.ViolationState, search.DeploymentID)), true).Return(([]*storage.Alert)(nil), nil)

	modified, err := suite.alertManager.AlertAndNotify(suite.ctx, nil, WithDeploymentID("did", false))
	suite.False(modified.Cardinality() > 0)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestGetAlertsByClusterAndResource() {
	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx,
		testutils.PredMatcher("query for violation state, cluster id and resource type", queryHasFields(search.ViolationState, search.ClusterID, search.ResourceType)), true,
	).Return(([]*storage.Alert)(nil), nil)

	modified, err := suite.alertManager.AlertAndNotify(suite.ctx, nil, WithLifecycleStage(storage.LifecycleStage_RUNTIME), WithClusterID("cid"), WithNamespace("nn"), WithResource("rn", storage.Alert_Resource_SECRETS))
	suite.False(modified.Cardinality() > 0)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestOnUpdatesWhenAlertsDoNotChange() {
	alerts := getAlerts()

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)
	// No updates should be attempted

	modified, err := suite.alertManager.AlertAndNotify(suite.ctx, alerts)
	suite.False(modified.Cardinality() > 0)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMarksOldAlertsResolved() {
	alerts := getAlerts()

	suite.alertsMock.EXPECT().MarkAlertsResolvedBatch(suite.ctx, alerts[0].GetId()).Return([]*storage.Alert{alerts[0]}, nil)

	// Unchanged alerts should not be updated.

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)
	// We should get a notification for the new alert.
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), alerts[0]).Return()

	// Make one of the alerts not appear in the current alerts.
	modified, err := suite.alertManager.AlertAndNotify(suite.ctx, alerts[1:])
	suite.True(modified.Cardinality() > 0)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestSendsNotificationsForNewAlerts() {
	alerts := getAlerts()

	// Only the new alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, alerts[0]).Return(nil)

	// We should get a notification for the new alert.
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), alerts[0]).Return()

	// Make one of the alerts not appear in the previous alerts.
	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts[1:], nil)

	modified, err := suite.alertManager.AlertAndNotify(suite.ctx, alerts)
	suite.True(modified.Cardinality() > 0)
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestNewResourceAlertIsAdded() {
	alerts := getResourceAlerts()
	newAlert := fixtures.GetResourceAlert()

	// Only the new alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, newAlert).Return(nil)

	// We should get a notification for the new alert.
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert).Return()

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeResourceAlerts() {
	alerts := getResourceAlerts()
	newAlert := alerts[0].CloneVT()
	newAlert.GetViolations()[0].SetMessage("new-violation")

	expectedMergedAlert := newAlert.CloneVT()
	expectedMergedAlert.SetViolations(append(expectedMergedAlert.GetViolations(), alerts[0].GetViolations()...))

	// Only the merged alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, protomock.GoMockMatcherEqualMessage(expectedMergedAlert)).Return(nil)

	// Updated alert should notify
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert).Return()

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeResourceAlertsNoNotify() {
	suite.T().Setenv("NOTIFY_EVERY_RUNTIME_EVENT", "false")
	alerts := getResourceAlerts()
	newAlert := alerts[0].CloneVT()
	newAlert.GetViolations()[0].SetMessage("new-violation")

	expectedMergedAlert := newAlert.CloneVT()
	expectedMergedAlert.SetViolations(append(expectedMergedAlert.GetViolations(), alerts[0].GetViolations()...))

	// Only the merged alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, protomock.GoMockMatcherEqualMessage(expectedMergedAlert)).Return(nil)

	// Updated alert should not notify

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeMultipleResourceAlerts() {
	alerts := getResourceAlerts()
	newAlert := alerts[0].CloneVT()
	newAlert.GetViolations()[0].SetMessage("new-violation")
	newAlert2 := alerts[0].CloneVT()
	newAlert2.GetViolations()[0].SetMessage("new-violation-2")

	// There will be two calls to Upsert
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, gomock.Any()).Return(nil)
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, gomock.Any()).Return(nil)

	// Updated alert should notify
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert).Return()
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert2).Return()

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert, newAlert2})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeResourceAlertsKeepsNewViolationsIfMoreThanMax() {
	alerts := getResourceAlerts()
	newAlert := alerts[0].CloneVT()
	newAlert.SetViolations(make([]*storage.Alert_Violation, maxRunTimeViolationsPerAlert))
	for i := 0; i < maxRunTimeViolationsPerAlert; i++ {
		av := &storage.Alert_Violation{}
		av.SetMessage(fmt.Sprintf("new-violation-%d", i))
		av.SetType(storage.Alert_Violation_K8S_EVENT)
		newAlert.GetViolations()[i] = av
	}

	expectedMergedAlert := newAlert.CloneVT()
	expectedMergedAlert.SetViolations(append(expectedMergedAlert.GetViolations(), alerts[0].GetViolations()...))
	expectedMergedAlert.SetViolations(expectedMergedAlert.GetViolations()[:maxRunTimeViolationsPerAlert])

	// Only the merged alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, protomock.GoMockMatcherEqualMessage(expectedMergedAlert)).Return(nil)

	// Updated alert should notify if set to
	if env.NotifyOnEveryRuntimeEvent() {
		suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert).Return()
	}

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeResourceAlertsKeepsNewViolationsIfMoreThanMaxNoNotify() {
	suite.T().Setenv("NOTIFY_EVERY_RUNTIME_EVENT", "false")
	alerts := getResourceAlerts()
	newAlert := alerts[0].CloneVT()
	newAlert.SetViolations(make([]*storage.Alert_Violation, maxRunTimeViolationsPerAlert))
	for i := 0; i < maxRunTimeViolationsPerAlert; i++ {
		av := &storage.Alert_Violation{}
		av.SetMessage(fmt.Sprintf("new-violation-%d", i))
		av.SetType(storage.Alert_Violation_K8S_EVENT)
		newAlert.GetViolations()[i] = av
	}

	expectedMergedAlert := newAlert.CloneVT()
	expectedMergedAlert.SetViolations(append(expectedMergedAlert.GetViolations(), alerts[0].GetViolations()...))
	expectedMergedAlert.SetViolations(expectedMergedAlert.GetViolations()[:maxRunTimeViolationsPerAlert])

	// Only the merged alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, protomock.GoMockMatcherEqualMessage(expectedMergedAlert)).Return(nil)

	// Updated alert should not notify

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeResourceAlertsOnlyKeepsMaxViolations() {
	alerts := getResourceAlerts()
	alerts[0].SetViolations(make([]*storage.Alert_Violation, maxRunTimeViolationsPerAlert))
	for i := 0; i < maxRunTimeViolationsPerAlert; i++ {
		av := &storage.Alert_Violation{}
		av.SetMessage(fmt.Sprintf("old-violation-%d", i))
		av.SetType(storage.Alert_Violation_K8S_EVENT)
		alerts[0].GetViolations()[i] = av
	}
	newAlert := alerts[0].CloneVT()
	newAlert.GetViolations()[0].SetMessage("new-violation")

	expectedMergedAlert := newAlert.CloneVT()

	// Only the merged alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, protomock.GoMockMatcherEqualMessage(expectedMergedAlert)).Return(nil)

	// Updated alert should notify if set to
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert).Return()

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestMergeResourceAlertsOnlyKeepsMaxViolationsNoNotify() {
	suite.T().Setenv("NOTIFY_EVERY_RUNTIME_EVENT", "false")
	alerts := getResourceAlerts()
	alerts[0].SetViolations(make([]*storage.Alert_Violation, maxRunTimeViolationsPerAlert))
	for i := 0; i < maxRunTimeViolationsPerAlert; i++ {
		av := &storage.Alert_Violation{}
		av.SetMessage(fmt.Sprintf("old-violation-%d", i))
		av.SetType(storage.Alert_Violation_K8S_EVENT)
		alerts[0].GetViolations()[i] = av
	}
	newAlert := alerts[0].CloneVT()
	newAlert.GetViolations()[0].SetMessage("new-violation")

	expectedMergedAlert := newAlert.CloneVT()

	// Only the merged alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, protomock.GoMockMatcherEqualMessage(expectedMergedAlert)).Return(nil)

	// Updated alert should not notify

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Add all the policies from the old alerts so that they aren't marked as stale
	for _, a := range alerts {
		suite.NoError(suite.policySet.UpsertPolicy(a.GetPolicy()))
	}
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
}

func (suite *AlertManagerTestSuite) TestOldResourceAlertAreMarkedAsResolvedWhenPolicyIsRemoved() {
	alerts := getResourceAlerts()
	newAlert := fixtures.GetResourceAlert()

	// Only the new alert will be updated.
	suite.alertsMock.EXPECT().UpsertAlert(suite.ctx, newAlert).Return(nil)

	// We should get a notifications for new alert
	suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), newAlert).Return()

	suite.alertsMock.EXPECT().SearchRawAlerts(suite.ctx, gomock.Any(), true).Return(alerts, nil)

	// Don't add any policies to simulate policies being deleted
	suite.runtimeDetectorMock.EXPECT().PolicySet().Return(suite.policySet).AnyTimes()

	ids := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		ids = append(ids, alert.GetId())
	}

	// Verify that the other alerts get marked as stale and that the notifier sends a notification for them
	suite.alertsMock.EXPECT().MarkAlertsResolvedBatch(suite.ctx, ids).Return(alerts, nil)

	for _, a := range alerts {
		suite.notifierMock.EXPECT().ProcessAlert(gomock.Any(), a).Return()
	}

	modifiedDeployments, err := suite.alertManager.AlertAndNotify(suite.ctx, []*storage.Alert{newAlert})
	suite.Equal(0, modifiedDeployments.Cardinality(), "no deployments should be modified when only resource alerts are provided")
	suite.NoError(err, "update should succeed")
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
				protoassert.Equal(t, c.expectedNew, c.new)
			}
		})
	}
}

func TestMergeRunTimeAlerts(t *testing.T) {
	for _, c := range []struct {
		desc           string
		old            *storage.Alert
		new            *storage.Alert
		expectedNew    *storage.Alert
		expectedOutput bool
	}{
		{
			desc: "dfdf",
			old: appendViolations(
				getFakeResourceRuntimeAlert(storage.Alert_Resource_SECRETS, "rn", "cid", "nid", "nn"),
				firstKubeEventViolation,
			),
			new: appendViolations(
				getFakeResourceRuntimeAlert(storage.Alert_Resource_SECRETS, "rn", "cid", "nid", "nn"),
				secondKubeEventViolation,
			),
			expectedNew: appendViolations(
				getFakeResourceRuntimeAlert(storage.Alert_Resource_SECRETS, "rn", "cid", "nid", "nn"),
				secondKubeEventViolation,
				firstKubeEventViolation,
			),
			expectedOutput: true,
		},
		{
			desc:           "Empty old alert; non-empty new alert",
			old:            getFakeRuntimeAlert(),
			new:            getFakeRuntimeAlert(yesterdayProcess),
			expectedNew:    appendViolations(getFakeRuntimeAlert(yesterdayProcess)),
			expectedOutput: true,
		},
		{
			desc:           "Empty old alert; non-empty new alert; again",
			old:            getFakeRuntimeAlert(),
			new:            getFakeRuntimeAlert(yesterdayProcess, nowProcess),
			expectedNew:    appendViolations(getFakeRuntimeAlert(yesterdayProcess, nowProcess)),
			expectedOutput: true,
		},
		{
			desc:           "No process; no event",
			old:            getFakeRuntimeAlert(),
			new:            getFakeRuntimeAlert(),
			expectedOutput: false,
		},
		{
			desc:           "No new process; no event",
			old:            getFakeRuntimeAlert(yesterdayProcess),
			new:            getFakeRuntimeAlert(),
			expectedOutput: false,
		},
		{
			desc:           "No process; no new event",
			old:            appendViolations(getFakeRuntimeAlert(), firstKubeEventViolation),
			new:            getFakeRuntimeAlert(),
			expectedOutput: false,
		},
		{
			desc:           "No process; new event",
			old:            getFakeRuntimeAlert(),
			new:            appendViolations(getFakeRuntimeAlert(), firstKubeEventViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(), firstKubeEventViolation),
			expectedOutput: true,
		},
		{
			desc:           "Equal process; no new event",
			old:            appendViolations(getFakeRuntimeAlert(yesterdayProcess), firstKubeEventViolation),
			new:            appendViolations(getFakeRuntimeAlert(yesterdayProcess)),
			expectedOutput: false,
		},
		{
			desc:           "Equal process; new event",
			old:            appendViolations(getFakeRuntimeAlert(yesterdayProcess), firstKubeEventViolation),
			new:            appendViolations(getFakeRuntimeAlert(yesterdayProcess), secondKubeEventViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(yesterdayProcess), secondKubeEventViolation, firstKubeEventViolation),
			expectedOutput: true,
		},
		{
			desc:           "New process; new event ",
			old:            appendViolations(getFakeRuntimeAlert(yesterdayProcess), firstKubeEventViolation),
			new:            appendViolations(getFakeRuntimeAlert(nowProcess), secondKubeEventViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(yesterdayProcess, nowProcess), secondKubeEventViolation, firstKubeEventViolation),
			expectedOutput: true,
		},
		{
			desc:           "New process; no new event ",
			old:            appendViolations(getFakeRuntimeAlert(yesterdayProcess), firstKubeEventViolation),
			new:            getFakeRuntimeAlert(nowProcess),
			expectedNew:    getFakeRuntimeAlert(yesterdayProcess, nowProcess),
			expectedOutput: true,
		},
		{
			desc:           "Many new process; many new events",
			old:            getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess),
			new:            appendViolations(getFakeRuntimeAlert(yesterdayProcess, nowProcess), firstKubeEventViolation, secondKubeEventViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess, nowProcess), firstKubeEventViolation, secondKubeEventViolation),
			expectedOutput: true,
		},
		{
			desc:           "No process; new network flow",
			old:            getFakeRuntimeAlert(),
			new:            appendViolations(getFakeRuntimeAlert(), firstNetworkFlowViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(), firstNetworkFlowViolation),
			expectedOutput: true,
		},
		{
			desc:           "Old process with old flow; new network flow",
			old:            appendViolations(getFakeRuntimeAlert(nowProcess), firstNetworkFlowViolation),
			new:            appendViolations(getFakeRuntimeAlert(nowProcess), secondNetworkFlowViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(nowProcess), secondNetworkFlowViolation, firstNetworkFlowViolation),
			expectedOutput: true,
		},
		{
			desc:           "Many new process; many new flow",
			old:            appendViolations(getFakeRuntimeAlert(twoDaysAgoProcess)),
			new:            appendViolations(getFakeRuntimeAlert(yesterdayProcess, nowProcess), firstNetworkFlowViolation, secondNetworkFlowViolation),
			expectedNew:    appendViolations(getFakeRuntimeAlert(twoDaysAgoProcess, yesterdayProcess, nowProcess), firstNetworkFlowViolation, secondNetworkFlowViolation),
			expectedOutput: true,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			out := mergeRunTimeAlerts(c.old, c.new)
			assert.Equal(t, c.expectedOutput, out)
			if c.expectedNew != nil {
				protoassert.Equal(t, c.expectedNew, c.new)
			}
		})
	}
}

func TestFindAlert(t *testing.T) {
	resourceAlerts := []*storage.Alert{getResourceAlerts()[0], fixtures.GetResourceAlert()}

	resourceAlertWithAltPolicy := getResourceAlerts()[0].CloneVT()
	resourceAlertWithAltPolicy.SetPolicy(getPolicies()[0].CloneVT())

	resourceAlertWithAltPolicyAndResource := getResourceAlerts()[1].CloneVT()
	resourceAlertWithAltPolicyAndResource.SetPolicy(getPolicies()[0].CloneVT())

	for _, c := range []struct {
		desc     string
		toFind   *storage.Alert
		alerts   []*storage.Alert
		expected *storage.Alert
	}{
		// ------ Deployment alerts
		{
			desc:     "Same policy, same deploy, Same state, Alert found",
			toFind:   getAlerts()[0],
			alerts:   getAlerts(),
			expected: getAlerts()[0],
		},
		{
			desc:     "Diff policy, Diff deploy, Same state, No alert found",
			toFind:   fixtures.GetAlert(),
			alerts:   getAlerts(),
			expected: nil,
		},
		// ------ Resource alerts
		{
			desc:     "Same policy, Same resource, Same state, Alert found",
			toFind:   getResourceAlerts()[0],
			alerts:   resourceAlerts,
			expected: getResourceAlerts()[0],
		},
		{
			desc:     "Diff policy, Same resource, Same state, No alert found",
			toFind:   resourceAlertWithAltPolicy,
			alerts:   resourceAlerts,
			expected: nil,
		},
		{
			desc:     "Diff policy, Diff resource, Same state, No alert found",
			toFind:   resourceAlertWithAltPolicyAndResource,
			alerts:   resourceAlerts,
			expected: nil,
		},
		{
			desc:     "Same policy, Diff resource (resource type), Same state, No alert found",
			toFind:   getResourceAlerts()[1],
			alerts:   resourceAlerts,
			expected: nil,
		},
		{
			desc:     "Same policy, Diff resource (resource name), Same state, No alert found",
			toFind:   getResourceAlerts()[2],
			alerts:   resourceAlerts,
			expected: nil,
		},
		{
			desc:     "Same policy, Diff resource (cluster), Same state, No alert found",
			toFind:   getResourceAlerts()[3],
			alerts:   resourceAlerts,
			expected: nil,
		},
		{
			desc:     "Same policy, Diff resource (namespace), Same state, No alert found",
			toFind:   getResourceAlerts()[4],
			alerts:   resourceAlerts,
			expected: nil,
		},
		// ------ Mixed case
		{
			desc:     "Deployment alert in a list of mixed alerts, Alert found",
			toFind:   getAlerts()[0],
			alerts:   append(getAlerts(), getResourceAlerts()...),
			expected: getAlerts()[0],
		},
		{
			desc:     "Resource alert in a list of mixed alerts, Alert found",
			toFind:   getResourceAlerts()[0],
			alerts:   append(getAlerts(), getResourceAlerts()...),
			expected: getResourceAlerts()[0],
		},
		{
			desc:     "Deployment alert in a list of resource alerts, No alert found",
			toFind:   getAlerts()[0],
			alerts:   getResourceAlerts(),
			expected: nil,
		},
		{
			desc:     "Resource alert in a list of deployment alerts, No alert found",
			toFind:   getResourceAlerts()[0],
			alerts:   getAlerts(),
			expected: nil,
		},
		{
			desc:     "Resource alert in a list of mixed alerts that share same policy, No alert found",
			toFind:   resourceAlertWithAltPolicy,
			alerts:   append(getAlerts(), resourceAlertWithAltPolicy),
			expected: resourceAlertWithAltPolicy,
		},
	} {
		t.Run(c.desc, func(t *testing.T) {
			found := findAlert(c.toFind, c.alerts)
			protoassert.Equal(t, c.expected, found)
		})
	}
}

//////////////////////////////////////
// TEST DATA
///////////////////////////////////////

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getAlerts() []*storage.Alert {
	alert := &storage.Alert{}
	alert.SetId("alert1")
	alert.SetPolicy(getPolicies()[0])
	alert.SetDeployment(proto.ValueOrDefault(getDeployments()[0]))
	alert.SetTime(protocompat.GetProtoTimestampFromSeconds(100))
	alert2 := &storage.Alert{}
	alert2.SetId("alert2")
	alert2.SetPolicy(getPolicies()[1])
	alert2.SetDeployment(proto.ValueOrDefault(getDeployments()[1]))
	alert2.SetTime(protocompat.GetProtoTimestampFromSeconds(200))
	alert3 := &storage.Alert{}
	alert3.SetId("alert3")
	alert3.SetPolicy(getPolicies()[2])
	alert3.SetDeployment(proto.ValueOrDefault(getDeployments()[2]))
	alert3.SetTime(protocompat.GetProtoTimestampFromSeconds(300))
	return []*storage.Alert{
		alert,
		alert2,
		alert3,
	}
}

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getDeployments() []*storage.Alert_Deployment {
	return []*storage.Alert_Deployment{
		storage.Alert_Deployment_builder{
			Name: "deployment1",
			Containers: []*storage.Alert_Deployment_Container{
				storage.Alert_Deployment_Container_builder{
					Image: storage.ContainerImage_builder{
						Name: storage.ImageName_builder{
							Tag:    "latest1",
							Remote: "stackrox/health",
						}.Build(),
					}.Build(),
				}.Build(),
			},
		}.Build(),
		storage.Alert_Deployment_builder{
			Name: "deployment2",
			Containers: []*storage.Alert_Deployment_Container{
				storage.Alert_Deployment_Container_builder{
					Image: storage.ContainerImage_builder{
						Name: storage.ImageName_builder{
							Tag:    "latest2",
							Remote: "stackrox/health",
						}.Build(),
					}.Build(),
				}.Build(),
			},
		}.Build(),
		storage.Alert_Deployment_builder{
			Name: "deployment3",
			Containers: []*storage.Alert_Deployment_Container{
				storage.Alert_Deployment_Container_builder{
					Image: storage.ContainerImage_builder{
						Name: storage.ImageName_builder{
							Tag:    "latest3",
							Remote: "stackrox/health",
						}.Build(),
					}.Build(),
				}.Build(),
			},
		}.Build(),
	}
}

// Policies are set up so that policy one is violated by deployment 1, 2 is violated by 2, etc.
func getPolicies() []*storage.Policy {
	return []*storage.Policy{
		storage.Policy_builder{
			Id:         "policy1",
			Name:       "latest1",
			Severity:   storage.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			PolicySections: []*storage.PolicySection{
				storage.PolicySection_builder{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						storage.PolicyGroup_builder{
							FieldName: fieldnames.ImageTag,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "latest1",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
			PolicyVersion: "1.1",
		}.Build(),
		storage.Policy_builder{
			Id:         "policy2",
			Name:       "latest2",
			Severity:   storage.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			PolicySections: []*storage.PolicySection{
				storage.PolicySection_builder{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						storage.PolicyGroup_builder{
							FieldName: fieldnames.ImageTag,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "latest2",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
		storage.Policy_builder{
			Id:         "policy3",
			Name:       "latest3",
			Severity:   storage.Severity_LOW_SEVERITY,
			Categories: []string{"Image Assurance", "Privileges Capabilities"},
			PolicySections: []*storage.PolicySection{
				storage.PolicySection_builder{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						storage.PolicyGroup_builder{
							FieldName: fieldnames.ImageTag,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "latest3",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
			PolicyVersion: "1.1",
		}.Build(),
	}
}

// Each alert is for a different resource where each resource after the 0th one is different in one property:
// type, name, cluster & namespace in that order. Everything else is the same
func getResourceAlerts() []*storage.Alert {
	return []*storage.Alert{
		storage.Alert_builder{
			Id:             "alert1",
			Policy:         fixtures.GetAuditLogEventSourcePolicy(),
			Resource:       proto.ValueOrDefault(getResources()[0]),
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Time:           protocompat.GetProtoTimestampFromSeconds(100),
			Violations:     []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "violation-alert-1", Type: storage.Alert_Violation_K8S_EVENT}.Build()},
		}.Build(),
		storage.Alert_builder{
			Id:             "alert2",
			Policy:         fixtures.GetAuditLogEventSourcePolicy(),
			Resource:       proto.ValueOrDefault(getResources()[1]),
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Time:           protocompat.GetProtoTimestampFromSeconds(200),
			Violations:     []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "violation-alert-2", Type: storage.Alert_Violation_K8S_EVENT}.Build()},
		}.Build(),
		storage.Alert_builder{
			Id:             "alert3",
			Policy:         fixtures.GetAuditLogEventSourcePolicy(),
			Resource:       proto.ValueOrDefault(getResources()[2]),
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Time:           protocompat.GetProtoTimestampFromSeconds(300),
			Violations:     []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "violation-alert-3", Type: storage.Alert_Violation_K8S_EVENT}.Build()},
		}.Build(),
		storage.Alert_builder{
			Id:             "alert4",
			Policy:         fixtures.GetAuditLogEventSourcePolicy(),
			Resource:       proto.ValueOrDefault(getResources()[3]),
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Time:           protocompat.GetProtoTimestampFromSeconds(400),
			Violations:     []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "violation-alert-4", Type: storage.Alert_Violation_K8S_EVENT}.Build()},
		}.Build(),
		storage.Alert_builder{
			Id:             "alert5",
			Policy:         fixtures.GetAuditLogEventSourcePolicy(),
			Resource:       proto.ValueOrDefault(getResources()[4]),
			LifecycleStage: storage.LifecycleStage_RUNTIME,
			Time:           protocompat.GetProtoTimestampFromSeconds(500),
			Violations:     []*storage.Alert_Violation{storage.Alert_Violation_builder{Message: "violation-alert-5", Type: storage.Alert_Violation_K8S_EVENT}.Build()},
		}.Build(),
	}
}

// Each resource after the 0th one is different in one property: type, name, cluster & namespace in that order
func getResources() []*storage.Alert_Resource {
	return []*storage.Alert_Resource{
		storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_SECRETS,
			Name:         "rez-name",
			ClusterId:    "cluster-id",
			ClusterName:  "prod cluster",
			Namespace:    "stackrox",
			NamespaceId:  "namespace-id",
		}.Build(),
		storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_CONFIGMAPS,
			Name:         "rez-name",
			ClusterId:    "cluster-id",
			ClusterName:  "prod cluster",
			Namespace:    "stackrox",
			NamespaceId:  "namespace-id",
		}.Build(),
		storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_SECRETS,
			Name:         "rez-name-alt",
			ClusterId:    "cluster-id",
			ClusterName:  "prod cluster",
			Namespace:    "stackrox",
			NamespaceId:  "namespace-id",
		}.Build(),
		storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_SECRETS,
			Name:         "rez-name",
			ClusterId:    "cluster-id-alt",
			ClusterName:  "prod cluster-alt",
			Namespace:    "stackrox",
			NamespaceId:  "namespace-id",
		}.Build(),
		storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_SECRETS,
			Name:         "rez-name",
			ClusterId:    "cluster-id",
			ClusterName:  "prod cluster",
			Namespace:    "stackrox-alt",
			NamespaceId:  "namespace-id-alt",
		}.Build(),
	}
}
