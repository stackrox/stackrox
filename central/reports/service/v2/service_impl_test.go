package v2

import (
	"context"
	"testing"

	reportConfigDSMocks "github.com/stackrox/rox/central/reportconfigurations/datastore/mocks"
	metadataDSMocks "github.com/stackrox/rox/central/reports/metadata/datastore/mocks"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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
	s := New(suite.reportMetadataStore, nil, nil, nil)
	repStatusResponse, err := s.GetReportStatus(suite.ctx, &id)
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
	s := serviceImpl{snapshotDatastore: suite.reportSnapshotStore}
	emptyQuery := &apiV2.RawQuery{Query: ""}
	req := &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	res, err := s.GetReportHistory(suite.ctx, req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetId(), "test_report")
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "",
		ReportParamQuery: emptyQuery,
	}

	_, err = s.GetReportHistory(suite.ctx, req)
	assert.Error(suite.T(), err)

	query := &apiV2.RawQuery{Query: "Report Name:test_report"}
	req = &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: query,
	}

	res, err = s.GetReportHistory(suite.ctx, req)
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

//func (suite *ReportServiceTestSuite) TestRunReport() {
//	cases := []struct{
//		desc string
//		req *apiV2.RunReportRequest
//		ctx context.Context
//		isError bool
//		resp *apiV2.RunReportResponse
//	}
//}
