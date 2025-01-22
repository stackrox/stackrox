package generator

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	checkResultsMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	remediationMocks "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/sender"
	ruleMocks "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
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
	formatter         *mocks.MockFormatter
	resultsAggregator *mocks.MockResultsAggregator
	reportSender      *mocks.MockReportSender
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
	s.formatter = mocks.NewMockFormatter(s.mockCtrl)
	s.resultsAggregator = mocks.NewMockResultsAggregator(s.mockCtrl)

	s.reportSender = mocks.NewMockReportSender(s.mockCtrl)

	s.reportGen = &complianceReportGeneratorImpl{
		checkResultsDS:         s.checkResultsDS,
		snapshotDS:             s.snapshotDS,
		profileDS:              s.profileDS,
		remediationDS:          s.remediationDS,
		complianceRuleDS:       s.ruleDS,
		notificationProcessor:  s.notifierProcessor,
		formatter:              s.formatter,
		resultsAggregator:      s.resultsAggregator,
		reportSender:           s.reportSender,
		senderResponseHandlers: make(map[string]stoppable[error]),
		newHandlerFn:           sender.NewAsyncResponseHandler[error],
	}
}

func TestComplianceReporting(t *testing.T) {
	suite.Run(t, new(ComplainceReportingTestSuite))
}

func (s *ComplainceReportingTestSuite) TestProcessReportRequest() {
	request := &report.Request{
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
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any()).Times(1)
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
				s.Require().Equal(storage.ComplianceOperatorReportStatus_GENERATED, snapshot.GetReportStatus().GetRunState())
				return errors.New("some error")
			})
		s.Require().Error(s.reportGen.ProcessReportRequest(request))
	})

	s.Run("Fail to notify", func() {
		s.reportGen.numberOfTriesOnEmailSend = 1
		wg := concurrency.NewWaitGroup(2)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any()).Times(1)
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
		s.reportSender.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1).
			DoAndReturn(func(_, _, _, _, _ any) <-chan error {
				errC := make(chan error)
				go func() {
					defer close(errC)
					errC <- errors.New("error")
				}()
				return errC
			})
		s.Require().NoError(s.reportGen.ProcessReportRequest(request))
		require.Eventually(s.T(), func() bool {
			return concurrency.WithLock1[bool](&s.reportGen.handlersMutex, func() bool {
				return len(s.reportGen.senderResponseHandlers) == 0
			})
		}, 500*time.Millisecond, 10*time.Millisecond)
		handleWaitGroup(s.T(), &wg, 500*time.Millisecond, "send email failure")
	})

	s.Run("Notify success", func() {
		s.reportGen.numberOfTriesOnEmailSend = 1
		wg := concurrency.NewWaitGroup(2)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any()).Times(1)
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
		s.reportSender.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1).
			DoAndReturn(func(_, _, _, _, _ any) <-chan error {
				errC := make(chan error)
				go func() {
					defer close(errC)
					errC <- nil
				}()
				return errC
			})
		s.Require().NoError(s.reportGen.ProcessReportRequest(request))
		require.Eventually(s.T(), func() bool {
			return concurrency.WithLock1[bool](&s.reportGen.handlersMutex, func() bool {
				return len(s.reportGen.senderResponseHandlers) == 0
			})
		}, 500*time.Millisecond, 10*time.Millisecond)
		handleWaitGroup(s.T(), &wg, 5*time.Second, "send email failure")
	})
}

func handleWaitGroup(t *testing.T, wg *concurrency.WaitGroup, timeout time.Duration, msg string) {
	select {
	case <-time.After(timeout):
		t.Errorf("timeout waiting for %s", msg)
		t.Fail()
	case <-wg.Done():
	}
}
