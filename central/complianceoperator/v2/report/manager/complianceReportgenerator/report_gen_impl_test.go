package complianceReportgenerator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	checkResultsMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	remediationMocks "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	ruleMocks "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ComplainceReportingTestSuite struct {
	suite.Suite
	mockCtrl          *gomock.Controller
	ctx               context.Context
	reportGen         *complianceReportGeneratorImpl
	snapshotDS        *snapshotMocks.MockDataStore
	checkResultsDS    *checkResultsMocks.MockDataStore
	profileDS         *profileMocks.MockDataStore
	remediationDS     *remediationMocks.MockDataStore
	ruleDS            *ruleMocks.MockDataStore
	notifierProcessor *notifierMocks.MockProcessor
}

func (s *ComplainceReportingTestSuite) SetupSuite() {
	s.T().Setenv(features.ScanScheduleReportJobs.EnvVar(), "true")
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.mockCtrl = gomock.NewController(s.T())
	s.snapshotDS = snapshotMocks.NewMockDataStore(s.mockCtrl)
	s.checkResultsDS = checkResultsMocks.NewMockDataStore(s.mockCtrl)
	s.profileDS = profileMocks.NewMockDataStore(s.mockCtrl)
	s.remediationDS = remediationMocks.NewMockDataStore(s.mockCtrl)
	s.ruleDS = ruleMocks.NewMockDataStore(s.mockCtrl)
	s.notifierProcessor = notifierMocks.NewMockProcessor(s.mockCtrl)

	s.reportGen = &complianceReportGeneratorImpl{
		checkResultsDS:        s.checkResultsDS,
		snapshotDS:            s.snapshotDS,
		profileDS:             s.profileDS,
		remediationDS:         s.remediationDS,
		complianceRuleDS:      s.ruleDS,
		notificationProcessor: s.notifierProcessor,
	}
}

func TestComplianceReporting(t *testing.T) {
	suite.Run(t, new(ComplainceReportingTestSuite))
}

func (s *ComplainceReportingTestSuite) TestFormatReport() {

	_, err := format(s.getReportData())
	s.Require().NoError(err)

}

func (s *ComplainceReportingTestSuite) TestProcessReportRequest() {
	request := &ComplianceReportRequest{
		ScanConfigID: "scan-config-1",
		SnapshotID:   "snapshot-1",
		ClusterIDs:   []string{"cluster-1"},
		Profiles:     []string{"profile-1"},
		Notifiers: []*storage.NotifierConfiguration{
			{
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "notifier-1",
						MailingLists: []string{"test@test.com"},
					},
				},
			},
		},
	}

	s.Run("GetSnapshots data store error", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(nil, false, errors.New("some error"))
		s.Require().Error(s.reportGen.ProcessReportRequest(request))
	})

	s.Run("Snapshot not found", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(nil, false, nil)
		s.Require().Error(s.reportGen.ProcessReportRequest(request))
	})

	s.Run("Fail to upsert Snapshot", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.checkResultsDS.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(request.ClusterIDs)).
			Return(nil)
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
				s.Require().Equal(storage.ComplianceOperatorReportStatus_GENERATED, snapshot.GetReportStatus().GetRunState())
				return errors.New("some error")
			})
		s.Require().Error(s.reportGen.ProcessReportRequest(request))
	})

	s.Run("Fail to grab the notifiers", func() {
		wg := concurrency.NewWaitGroup(2)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.checkResultsDS.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(request.ClusterIDs)).
			Return(nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					s.Require().Equal(storage.ComplianceOperatorReportStatus_GENERATED, snapshot.GetReportStatus().GetRunState())
					wg.Add(-1)
					return nil
				}),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					s.Require().Equal(storage.ComplianceOperatorReportStatus_FAILURE, snapshot.GetReportStatus().GetRunState())
					wg.Add(-1)
					return nil
				}),
		)
		s.notifierProcessor.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Times(len(request.Notifiers)).
			Return(nil)
		s.Require().NoError(s.reportGen.ProcessReportRequest(request))
		handleWaitGroup(s.T(), &wg, 500*time.Millisecond, "send email failure")
	})

	s.Run("Fail to notify", func() {
		s.reportGen.numberOfTriesOnEmailSend = 1
		wg := concurrency.NewWaitGroup(3)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.checkResultsDS.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(request.ClusterIDs)).
			Return(nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					s.Require().Equal(storage.ComplianceOperatorReportStatus_GENERATED, snapshot.GetReportStatus().GetRunState())
					wg.Add(-1)
					return nil
				}),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					s.Require().Equal(storage.ComplianceOperatorReportStatus_FAILURE, snapshot.GetReportStatus().GetRunState())
					wg.Add(-1)
					return nil
				}),
		)
		s.notifierProcessor.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Times(len(request.Notifiers)).
			Return(&fakeNotifierAlwaysFail{&wg})
		s.Require().NoError(s.reportGen.ProcessReportRequest(request))
		handleWaitGroup(s.T(), &wg, 500*time.Millisecond, "send email failure")
	})

	s.Run("Notify success", func() {
		s.reportGen.numberOfTriesOnEmailSend = 1
		wg := concurrency.NewWaitGroup(3)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.checkResultsDS.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(request.ClusterIDs)).
			Return(nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					s.Require().Equal(storage.ComplianceOperatorReportStatus_GENERATED, snapshot.GetReportStatus().GetRunState())
					wg.Add(-1)
					return nil
				}),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					s.Require().Equal(storage.ComplianceOperatorReportStatus_DELIVERED, snapshot.GetReportStatus().GetRunState())
					wg.Add(-1)
					return nil
				}),
		)
		s.notifierProcessor.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Times(len(request.Notifiers)).
			Return(&fakeNotifierAlwaysSuccess{&wg})
		s.Require().NoError(s.reportGen.ProcessReportRequest(request))
		handleWaitGroup(s.T(), &wg, 5*time.Second, "send email failure")
	})
}

func (s *ComplainceReportingTestSuite) getReportData() map[string][]*ResultRow {
	results := make(map[string][]*ResultRow)
	results["cluster1"] = []*ResultRow{{
		ClusterName: "test_cluster1",
		CheckName:   "test_check1",
		Profile:     "test_profile1",
		ControlRef:  "test_control_ref1",
		Description: "description1",
		Status:      "Pass",
		Remediation: "remediation1",
	},
		{
			ClusterName: "test_cluster2",
			CheckName:   "test_check2",
			Profile:     "test_profile2",
			ControlRef:  "test_control_ref2",
			Description: "description2",
			Status:      "Fail",
			Remediation: "remediation2",
		},
	}
	return results
}

func handleWaitGroup(t *testing.T, wg *concurrency.WaitGroup, timeout time.Duration, msg string) {
	select {
	case <-time.After(timeout):
		t.Errorf("timeout waiting for %s", msg)
		t.Fail()
	case <-wg.Done():
	}
}

type fakeNotifierAlwaysSuccess struct {
	wg *concurrency.WaitGroup
}

func (f *fakeNotifierAlwaysSuccess) Close(_ context.Context) error {
	return nil
}
func (f *fakeNotifierAlwaysSuccess) ProtoNotifier() *storage.Notifier {
	return nil
}
func (f *fakeNotifierAlwaysSuccess) Test(_ context.Context) *notifiers.NotifierError {
	return &notifiers.NotifierError{}
}

func (f *fakeNotifierAlwaysSuccess) ReportNotify(_ context.Context, _ *bytes.Buffer, _ []string, _, _, _ string) error {
	f.wg.Add(-1)
	return nil
}

type fakeNotifierAlwaysFail struct {
	wg *concurrency.WaitGroup
}

func (f *fakeNotifierAlwaysFail) Close(_ context.Context) error {
	return nil
}
func (f *fakeNotifierAlwaysFail) ProtoNotifier() *storage.Notifier {
	return nil
}
func (f *fakeNotifierAlwaysFail) Test(_ context.Context) *notifiers.NotifierError {
	return &notifiers.NotifierError{}
}

func (f *fakeNotifierAlwaysFail) ReportNotify(_ context.Context, _ *bytes.Buffer, _ []string, _, _, _ string) error {
	f.wg.Add(-1)
	return errors.New("some error")
}
