package v2

import (
	"context"
	"io"
	"testing"

	"github.com/pkg/errors"
	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	reportConfigDSMocks "github.com/stackrox/rox/central/reportconfigurations/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestReportService(t *testing.T) {
	suite.Run(t, new(ReportServiceTestSuite))
}

type ReportServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                 context.Context
	reportConfigStore   *reportConfigDSMocks.MockDataStore
	reportSnapshotStore *reportSnapshotDSMocks.MockDataStore
	collectionStore     *collectionDSMocks.MockDataStore
	notifierStore       *notifierDSMocks.MockDataStore
	blobStore           *blobDSMocks.MockDatastore
	scheduler           *schedulerMocks.MockScheduler
	service             Service
}

func (suite *ReportServiceTestSuite) SetupSuite() {
	suite.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		suite.T().Skip("Skip test when reporting enhancements are disabled")
		suite.T().SkipNow()
	}

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.ctx = sac.WithAllAccess(context.Background())
	suite.reportConfigStore = reportConfigDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.reportSnapshotStore = reportSnapshotDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.collectionStore = collectionDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.notifierStore = notifierDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.scheduler = schedulerMocks.NewMockScheduler(suite.mockCtrl)
	suite.blobStore = blobDSMocks.NewMockDatastore(suite.mockCtrl)
	suite.service = New(suite.reportConfigStore, suite.reportSnapshotStore, suite.collectionStore, suite.notifierStore, suite.scheduler, suite.blobStore)
}

func (suite *ReportServiceTestSuite) TestGetReportStatus() {
	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	snapshot := &storage.ReportSnapshot{
		ReportId:     "test_report",
		ReportStatus: status,
	}

	suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(snapshot, true, nil)
	id := apiV2.ResourceByID{
		Id: "test_report",
	}
	repStatusResponse, err := suite.service.GetReportStatus(suite.ctx, &id)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), repStatusResponse.Status.GetErrorMsg(), status.GetErrorMsg())
}

func (suite *ReportServiceTestSuite) TestGetReportHistory() {
	reportSnapshot := &storage.ReportSnapshot{
		ReportId:              "test_report",
		ReportConfigurationId: "test_report_config",
		Name:                  "Report",
		ReportStatus: &storage.ReportStatus{
			ErrorMsg:                 "Error msg",
			ReportNotificationMethod: 1,
		},
	}

	suite.reportSnapshotStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	emptyQuery := &apiV2.RawQuery{Query: ""}
	req := &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	res, err := suite.service.GetReportHistory(suite.ctx, req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportJobId(), "test_report")
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "",
		ReportParamQuery: emptyQuery,
	}

	_, err = suite.service.GetReportHistory(suite.ctx, req)
	assert.Error(suite.T(), err)

	query := &apiV2.RawQuery{Query: "Report Name:test_report"}
	req = &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: query,
	}

	res, err = suite.service.GetReportHistory(suite.ctx, req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportJobId(), "test_report")
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")
}

func (suite *ReportServiceTestSuite) TestAuthz() {
	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	snapshot := &storage.ReportSnapshot{
		ReportId:     "test_report",
		ReportStatus: status,
	}
	snapshotDS := reportSnapshotDSMocks.NewMockDataStore(suite.mockCtrl)
	snapshotDS.EXPECT().Get(gomock.Any(), gomock.Any()).Return(snapshot, true, nil).AnyTimes()
	metadataSlice := []*storage.ReportSnapshot{snapshot}
	snapshotDS.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return(metadataSlice, nil).AnyTimes()
	s := serviceImpl{snapshotDatastore: snapshotDS}
	testutils.AssertAuthzWorks(suite.T(), &s)
}

func (suite *ReportServiceTestSuite) TestRunReport() {
	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiers()
	notifierIDs := make([]string, 0, len(reportConfig.GetNotifiers()))
	notifiers := make([]*storage.Notifier, 0, len(reportConfig.GetNotifiers()))
	for _, nc := range reportConfig.GetNotifiers() {
		notifierIDs = append(notifierIDs, nc.GetEmailConfig().GetNotifierId())
		notifiers = append(notifiers, &storage.Notifier{
			Id:   nc.GetEmailConfig().GetNotifierId(),
			Name: nc.GetEmailConfig().GetNotifierId(),
		})
	}
	collection := &storage.ResourceCollection{
		Id: reportConfig.GetResourceScope().GetCollectionId(),
	}

	user := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	userContext := authn.ContextWithIdentity(suite.ctx, mockID, suite.T())

	testCases := []struct {
		desc    string
		req     *apiV2.RunReportRequest
		ctx     context.Context
		mockGen func()
		isError bool
		resp    *apiV2.RunReportResponse
	}{
		{
			desc: "Report config ID empty",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           "",
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx:     suite.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report config not found",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Collection not found",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				suite.collectionStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(nil, false, nil)
			},
			isError: true,
		},
		{
			desc: "One of the notifiers not found",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				suite.collectionStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				suite.notifierStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return([]*storage.Notifier{notifiers[0]}, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Successful submission; Notification method email",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				suite.collectionStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				suite.notifierStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return(notifiers, nil).Times(1)
				suite.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), false).
					Return("reportID", nil).Times(1)
			},
			isError: false,
			resp: &apiV2.RunReportResponse{
				ReportConfigId: reportConfig.Id,
				ReportId:       "reportID",
			},
		},
		{
			desc: "Successful submission; Notification method download",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				suite.collectionStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				suite.notifierStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return(notifiers, nil).Times(1)
				suite.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), false).
					Return("reportID", nil).Times(1)
			},
			isError: false,
			resp: &apiV2.RunReportResponse{
				ReportConfigId: reportConfig.Id,
				ReportId:       "reportID",
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.mockGen()
			response, err := suite.service.RunReport(tc.ctx, tc.req)
			if tc.isError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.resp, response)
			}
		})
	}
}

func (suite *ReportServiceTestSuite) TestCancelReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.ReportId = uuid.NewV4().String()
	reportSnapshot.ReportStatus.RunState = storage.ReportStatus_WAITING
	user := reportSnapshot.GetRequester()

	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	userContext := authn.ContextWithIdentity(suite.ctx, mockID, suite.T())

	testCases := []struct {
		desc    string
		req     *apiV2.ResourceByID
		ctx     context.Context
		mockGen func()
		isError bool
	}{
		{
			desc: "Empty Report ID",
			req: &apiV2.ResourceByID{
				Id: "",
			},
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx:     suite.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report requester id and cancelling user id mismatch",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.Requester = &storage.SlimUser{
					Id:   reportSnapshot.Requester.Id + "-1",
					Name: reportSnapshot.Requester.Name + "-1",
				}
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report is already generated",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_SUCCESS
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report already in PREPARING state",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_PREPARING
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler error while cancelling request",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.scheduler.EXPECT().CancelReportRequest(gomock.Any(), gomock.Any()).
					Return(false, errors.New("Datastore error")).
					Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler couldn't find report ID in queue",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.scheduler.EXPECT().CancelReportRequest(gomock.Any(), gomock.Any()).
					Return(false, nil).
					Times(1)
			},
			isError: true,
		},
		{
			desc: "Request cancelled",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.scheduler.EXPECT().CancelReportRequest(gomock.Any(), gomock.Any()).
					Return(true, nil).
					Times(1)
			},
			isError: false,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.mockGen()
			response, err := suite.service.CancelReport(tc.ctx, tc.req)
			if tc.isError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(&apiV2.Empty{}, response)
			}
		})
	}
}

func (suite *ReportServiceTestSuite) TestDownloadReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.ReportId = uuid.NewV4().String()
	reportSnapshot.ReportConfigurationId = uuid.NewV4().String()
	reportSnapshot.ReportStatus.RunState = storage.ReportStatus_SUCCESS
	reportSnapshot.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	user := reportSnapshot.GetRequester()
	blob, blobData := fixtures.GetBlobWithData()

	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	blobName := common.GetReportBlobPath(reportSnapshot.GetReportId(), reportSnapshot.GetReportConfigurationId())

	userContext := authn.ContextWithIdentity(suite.ctx, mockID, suite.T())
	testCases := []struct {
		desc    string
		req     *apiV2.DownloadReportRequest
		ctx     context.Context
		mockGen func()
		isError bool
	}{
		{
			desc: "Empty Report ID",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: "",
			},
			ctx:     userContext,
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx:     suite.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Download requester id and report requester user id mismatch",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.Requester = &storage.SlimUser{
					Id:   reportSnapshot.Requester.Id + "-1",
					Name: reportSnapshot.Requester.Name + "-1",
				}
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report was not generated",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_EMAIL
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report is not ready yet",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_PREPARING
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Blob get error",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.blobStore.EXPECT().Get(gomock.Any(), blobName, gomock.Any()).Times(1).Return(nil, false, errors.New(""))
			},
			isError: true,
		},
		{
			desc: "Blob does not exist",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.blobStore.EXPECT().Get(gomock.Any(), blobName, gomock.Any()).Times(1).Return(nil, false, nil)
			},
			isError: true,
		},
		{
			desc: "Report downloaded",
			req: &apiV2.DownloadReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.blobStore.EXPECT().Get(gomock.Any(), blobName, gomock.Any()).Times(1).DoAndReturn(
					func(_ context.Context, _ string, writer io.Writer) (*storage.Blob, bool, error) {
						c, err := writer.Write(blobData.Bytes())
						suite.NoError(err)
						suite.Equal(c, blobData.Len())
						return blob, true, nil
					})
			},
			isError: false,
		},
	}
	for _, tc := range testCases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			if tc.mockGen != nil {
				tc.mockGen()
			}
			response, err := suite.service.DownloadReport(tc.ctx, tc.req)
			if tc.isError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(&apiV2.DownloadReportResponse{Data: blobData.Bytes()}, response)
			}
		})
	}

}

func (suite *ReportServiceTestSuite) TestDeleteReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.ReportId = uuid.NewV4().String()
	reportSnapshot.ReportConfigurationId = uuid.NewV4().String()
	reportSnapshot.ReportStatus.RunState = storage.ReportStatus_SUCCESS
	reportSnapshot.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	user := reportSnapshot.GetRequester()

	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	blobName := common.GetReportBlobPath(reportSnapshot.GetReportId(), reportSnapshot.GetReportConfigurationId())

	userContext := authn.ContextWithIdentity(suite.ctx, mockID, suite.T())
	testCases := []struct {
		desc    string
		req     *apiV2.DeleteReportRequest
		ctx     context.Context
		mockGen func()
		isError bool
	}{
		{
			desc: "Empty Report ID",
			req: &apiV2.DeleteReportRequest{
				ReportJobId: "",
			},
			ctx:     userContext,
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.DeleteReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx:     suite.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: &apiV2.DeleteReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Delete requester user id and report requester user id mismatch",
			req: &apiV2.DeleteReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.Requester = &storage.SlimUser{
					Id:   reportSnapshot.Requester.Id + "-1",
					Name: reportSnapshot.Requester.Name + "-1",
				}
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Delete blob failed",
			req: &apiV2.DeleteReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.blobStore.EXPECT().Delete(gomock.Any(), blobName).Times(1).Return(errors.New(""))
			},
			isError: true,
		},
		{
			desc: "Report deleted",
			req: &apiV2.DeleteReportRequest{
				ReportJobId: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportSnapshotStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				suite.blobStore.EXPECT().Delete(gomock.Any(), blobName).Times(1).Return(nil)
			},
			isError: false,
		},
	}
	for _, tc := range testCases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			if tc.mockGen != nil {
				tc.mockGen()
			}
			response, err := suite.service.DeleteReport(tc.ctx, tc.req)
			if tc.isError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(&apiV2.Empty{}, response)
			}
		})
	}

}
