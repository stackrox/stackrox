package v2

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	reportConfigDSMocks "github.com/stackrox/rox/central/reportconfigurations/datastore/mocks"
	metadataDSMocks "github.com/stackrox/rox/central/reports/metadata/datastore/mocks"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
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
	reportMetadataStore *metadataDSMocks.MockDataStore
	reportConfigStore   *reportConfigDSMocks.MockDataStore
	reportSnapshotStore *reportSnapshotDSMocks.MockDataStore
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
	suite.reportMetadataStore = metadataDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.reportConfigStore = reportConfigDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.reportSnapshotStore = reportSnapshotDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.scheduler = schedulerMocks.NewMockScheduler(suite.mockCtrl)
	suite.service = New(suite.reportMetadataStore, suite.reportConfigStore, suite.reportSnapshotStore, suite.scheduler)
}

func (suite *ReportServiceTestSuite) TestGetReportStatus() {
	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	metadata := &storage.ReportMetadata{
		ReportId:     "test_report",
		ReportStatus: status,
	}

	suite.reportMetadataStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(metadata, true, nil)
	id := apiV2.ResourceByID{
		Id: "test_report",
	}
	repStatusResponse, err := suite.service.GetReportStatus(suite.ctx, &id)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), repStatusResponse.Status.GetErrorMsg(), status.GetErrorMsg())
}

func (suite *ReportServiceTestSuite) TestGetReportHistory() {
	reportSnapshot := &storage.ReportSnapshot{
		ReportId: "test_report",
		Name:     "Report",
		ReportStatus: &storage.ReportStatus{
			ErrorMsg:                 "Error msg",
			ReportNotificationMethod: 1,
		},
		ReportConfigurationId: "test_report",
	}

	suite.reportSnapshotStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	emptyQuery := &apiV2.RawQuery{Query: ""}
	req := &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	res, err := suite.service.GetReportHistory(suite.ctx, req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetId(), "test_report")
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
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetId(), "test_report")
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")
}

func (suite *ReportServiceTestSuite) TestAuthz() {
	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	metadata := &storage.ReportMetadata{
		ReportId:     "test_report",
		ReportStatus: status,
	}
	metadataDS := metadataDSMocks.NewMockDataStore(suite.mockCtrl)
	metadataDS.EXPECT().Get(gomock.Any(), gomock.Any()).Return(metadata, true, nil).AnyTimes()
	metadataSlice := []*storage.ReportMetadata{metadata}
	metadataDS.EXPECT().SearchReportMetadatas(gomock.Any(), gomock.Any()).Return(metadataSlice, nil).AnyTimes()
	s := serviceImpl{metadataDatastore: metadataDS}
	testutils.AssertAuthzWorks(suite.T(), &s)
}

func (suite *ReportServiceTestSuite) TestRunReport() {
	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiers()

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
				suite.reportConfigStore.EXPECT().GetReportConfiguration(userContext, reportConfig.Id).
					Return(nil, false, nil).Times(1)
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
				suite.reportConfigStore.EXPECT().GetReportConfiguration(userContext, reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				reportReq := getReportRequest(user, reportConfig, storage.ReportStatus_EMAIL)
				suite.scheduler.EXPECT().SubmitReportRequest(reportReq, false).
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
				suite.reportConfigStore.EXPECT().GetReportConfiguration(userContext, reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				reportReq := getReportRequest(user, reportConfig, storage.ReportStatus_DOWNLOAD)
				suite.scheduler.EXPECT().SubmitReportRequest(reportReq, false).
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
	reportMetadata := fixtures.GetReportMetadata()
	reportMetadata.ReportStatus.RunState = storage.ReportStatus_WAITING
	user := reportMetadata.GetRequester()

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
				Id: reportMetadata.GetReportId(),
			},
			ctx:     suite.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report requester id and cancelling user id mismatch",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				metadata := reportMetadata.Clone()
				metadata.Requester = &storage.SlimUser{
					Id:   reportMetadata.Requester.Id + "-1",
					Name: reportMetadata.Requester.Name + "-1",
				}
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(metadata, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report ID is already generated",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				metadata := reportMetadata.Clone()
				metadata.ReportStatus.RunState = storage.ReportStatus_SUCCESS
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(metadata, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report already in PREPARING state",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				metadata := reportMetadata.Clone()
				metadata.ReportStatus.RunState = storage.ReportStatus_PREPARING
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(metadata, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler error while cancelling request",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(reportMetadata, true, nil).Times(1)
				suite.scheduler.EXPECT().CancelReportRequest(gomock.Any()).
					Return(false, errors.New("Datastore error")).
					Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler couldn't find report ID in queue",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(reportMetadata, true, nil).Times(1)
				suite.scheduler.EXPECT().CancelReportRequest(gomock.Any()).
					Return(false, nil).
					Times(1)
			},
			isError: true,
		},
		{
			desc: "Request cancelled",
			req: &apiV2.ResourceByID{
				Id: reportMetadata.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				suite.reportMetadataStore.EXPECT().Get(userContext, reportMetadata.GetReportId()).
					Return(reportMetadata, true, nil).Times(1)
				suite.scheduler.EXPECT().CancelReportRequest(gomock.Any()).
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

func getReportRequest(user *storage.SlimUser, config *storage.ReportConfiguration,
	notificationMethod storage.ReportStatus_NotificationMethod) *reportGen.ReportRequest {
	return &reportGen.ReportRequest{
		ReportConfig: config,
		ReportMetadata: &storage.ReportMetadata{
			ReportConfigId: config.Id,
			Requester:      user,
			ReportStatus: &storage.ReportStatus{
				RunState:                 storage.ReportStatus_WAITING,
				ReportRequestType:        storage.ReportStatus_ON_DEMAND,
				ReportNotificationMethod: notificationMethod,
			},
		},
	}
}
