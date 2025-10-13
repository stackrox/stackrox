package generator

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	blobMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
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
	"github.com/stackrox/rox/pkg/sac/resources"
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
	blobDS            *blobMocks.MockDatastore
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
	s.blobDS = blobMocks.NewMockDatastore(s.mockCtrl)
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
		blobStore:              s.blobDS,
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
	scanConfigName := "scan-test"
	request := &report.Request{
		Ctx:            context.Background(),
		ScanConfigID:   "scan-config-1",
		ScanConfigName: scanConfigName,
		SnapshotID:     "snapshot-1",
		ClusterIDs:     []string{"cluster-1"},
		Profiles:       []string{"profile-1"},
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
		ClusterData: map[string]*report.ClusterData{
			"cluster-1": {},
		},
	}

	s.Run("GetSnapshots data store error", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(nil, false, errors.New("some error"))
		err := s.reportGen.ProcessReportRequest(request)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToRetrieveSnapshotStr)
	})

	s.Run("Snapshot not found", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(nil, false, nil)
		err := s.reportGen.ProcessReportRequest(request)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToFindSnapshotStr)
	})

	s.Run("FormatCSVReport error", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("some error"))
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
			gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
				return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
			})).Times(1).Return(nil)
		err := s.reportGen.ProcessReportRequest(request)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), fmt.Sprintf(errUnableToGenerateReportFmt, scanConfigName))
	})

	s.Run("FormatCSVReport error and upsert snapshot error", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("some error"))
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
			gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
				return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
			})).Times(1).Return(errors.New("some error"))
		err := s.reportGen.ProcessReportRequest(request)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnGenerationFailureStr)
	})

	s.Run("Fail to upsert Snapshot (generated)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
			gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
				return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
			})).Times(1).Return(errors.New("some error"))
		err := s.reportGen.ProcessReportRequest(request)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnGenerationSuccessStr)
	})

	s.Run("Fail to upsert Snapshot (partial error for email)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
			gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
				return storage.ComplianceOperatorReportStatus_PARTIAL_SCAN_ERROR_EMAIL == target.GetReportStatus().GetRunState()
			})).Times(1).Return(errors.New("some error"))
		err := s.reportGen.ProcessReportRequest(newFakeRequestWithFailedCluster())
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnGenerationSuccessStr)
	})

	s.Run("Fail to upsert Snapshot (partial error for download)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
			gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
				return storage.ComplianceOperatorReportStatus_PARTIAL_SCAN_ERROR_DOWNLOAD == target.GetReportStatus().GetRunState()
			})).Times(1).Return(errors.New("some error"))
		req := newFakeDownloadRequest()
		req.ClusterData = map[string]*report.ClusterData{
			"cluster-2": {
				FailedInfo: &report.FailedCluster{},
			},
		}
		req.NumFailedClusters = 1
		err := s.reportGen.ProcessReportRequest(req)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnGenerationSuccessStr)
	})

	s.Run("Fail to upsert Snapshot (all clusters failed)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
			gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
				return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
			})).Times(1).Return(errors.New("some error"))
		req := newFakeRequestWithFailedCluster()
		req.ClusterData["cluster-1"] = &report.ClusterData{
			FailedInfo: &report.FailedCluster{},
		}
		req.NumFailedClusters = 2
		err := s.reportGen.ProcessReportRequest(req)
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnGenerationSuccessStr)
	})

	s.Run("Fail saving report data (nil data)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
		)
		err := s.reportGen.ProcessReportRequest(newFakeDownloadRequest())
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToSaveTheReportBlobStr)
	})

	s.Run("Fail saving report data (blob upsert error)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1).Return(&bytes.Buffer{}, nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
			s.blobDS.EXPECT().Upsert(gomock.Cond[context.Context](func(ctx context.Context) bool {
				return validateBlobContext(ctx)
			}), gomock.Any(), gomock.Any()).Times(1).Return(errors.New("some error")),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
		)
		err := s.reportGen.ProcessReportRequest(newFakeDownloadRequest())
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToSaveTheReportBlobStr)
	})

	s.Run("Fail saving report data (blob and snapshot upsert error)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1).Return(&bytes.Buffer{}, nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
			s.blobDS.EXPECT().Upsert(gomock.Cond[context.Context](func(ctx context.Context) bool {
				return validateBlobContext(ctx)
			}), gomock.Any(), gomock.Any()).Times(1).Return(errors.New("some error")),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
				})).Times(1).Return(errors.New("some error")),
		)
		err := s.reportGen.ProcessReportRequest(newFakeDownloadRequest())
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnBlobFailureStr)
	})

	s.Run("Saving report data success (snapshot upsert error)", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1).Return(&bytes.Buffer{}, nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
			s.blobDS.EXPECT().Upsert(gomock.Cond[context.Context](func(ctx context.Context) bool {
				return validateBlobContext(ctx)
			}), gomock.Any(), gomock.Any()).Times(1).Return(nil),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState() &&
						target.GetReportStatus().GetCompletedAt() != nil
				})).Times(1).Return(errors.New("some error")),
		)
		err := s.reportGen.ProcessReportRequest(newFakeDownloadRequest())
		s.Require().Error(err)
		s.Assert().Contains(err.Error(), errUnableToUpdateSnapshotOnBlobSuccessStr)
	})

	s.Run("Saving report data success", func() {
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1).Return(&bytes.Buffer{}, nil)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).Return(nil),
			s.blobDS.EXPECT().Upsert(gomock.Cond[context.Context](func(ctx context.Context) bool {
				return validateBlobContext(ctx)
			}), gomock.Any(), gomock.Any()).Times(1).Return(nil),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState() &&
						target.GetReportStatus().GetCompletedAt() != nil
				})).Times(1).Return(nil),
		)
		s.Require().NoError(s.reportGen.ProcessReportRequest(newFakeDownloadRequest()))
	})

	s.Run("Fail to notify", func() {
		s.reportGen.numberOfTriesOnEmailSend = 1
		wg := concurrency.NewWaitGroup(2)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(&storage.ComplianceOperatorReportSnapshotV2{
				ReportStatus: &storage.ComplianceOperatorReportStatus{},
			}, true, nil)
		s.resultsAggregator.EXPECT().GetReportData(gomock.Any()).Times(1).Return(&report.Results{})
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					wg.Add(-1)
					return nil
				}),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_FAILURE == target.GetReportStatus().GetRunState()
				})).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
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
		s.formatter.EXPECT().FormatCSVReport(gomock.Any(), gomock.Any()).Times(1)
		gomock.InOrder(
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_GENERATED == target.GetReportStatus().GetRunState()
				})).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
					wg.Add(-1)
					return nil
				}),
			s.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(),
				gomock.Cond[*storage.ComplianceOperatorReportSnapshotV2](func(target *storage.ComplianceOperatorReportSnapshotV2) bool {
					return storage.ComplianceOperatorReportStatus_DELIVERED == target.GetReportStatus().GetRunState()
				})).Times(1).
				DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
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

func newFakeDownloadRequest() *report.Request {
	return &report.Request{
		Ctx:                context.Background(),
		ScanConfigID:       "scan-config-1",
		SnapshotID:         "snapshot-1",
		ClusterIDs:         []string{"cluster-1", "cluster-2"},
		Profiles:           []string{"profile-1"},
		NotificationMethod: storage.ComplianceOperatorReportStatus_DOWNLOAD,
	}
}

func newFakeRequestWithFailedCluster() *report.Request {
	return &report.Request{
		ScanConfigID: "scan-config-1",
		SnapshotID:   "snapshot-1",
		ClusterIDs:   []string{"cluster-1", "cluster-2"},
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
		ClusterData: map[string]*report.ClusterData{
			"cluster-1": {},
			"cluster-2": {
				FailedInfo: &report.FailedCluster{},
			},
		},
		NumFailedClusters: 1,
	}
}

func validateBlobContext(ctx context.Context) bool {
	scopeChecker := sac.ForResource(resources.Administration)
	return scopeChecker.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed()
}

func handleWaitGroup(t *testing.T, wg *concurrency.WaitGroup, timeout time.Duration, msg string) {
	select {
	case <-time.After(timeout):
		t.Errorf("timeout waiting for %s", msg)
		t.Fail()
	case <-wg.Done():
	}
}
