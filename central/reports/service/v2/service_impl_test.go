package v2

import (
	"context"
	"testing"

	metadataDSMocks "github.com/stackrox/rox/central/reports/metadata/datastore/mocks"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
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

	reportDS          *metadataDSMocks.MockDataStore
	snapshotDS        *reportSnapshotDSMocks.MockDataStore
	collectionService Service
}

func (suite *ReportServiceTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.reportDS = metadataDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.snapshotDS = reportSnapshotDSMocks.NewMockDataStore(suite.mockCtrl)

}
func (suite *ReportServiceTestSuite) TestGetReportStatus() {

	suite.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		suite.T().Skip("Skip test when reporting enhancements are disabled")
		suite.T().SkipNow()
	}
	ctx := context.Background()

	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	metadata := &storage.ReportMetadata{
		ReportId:     "test_report",
		ReportStatus: status,
	}

	suite.reportDS.EXPECT().Get(gomock.Any(), gomock.Any()).Return(metadata, true, nil)
	id := apiV2.ResourceByID{
		Id: "test_report",
	}
	s := New(suite.reportDS, nil, nil, nil)
	repStatusResponse, err := s.GetReportStatus(ctx, &id)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), repStatusResponse.Status.GetErrorMsg(), status.GetErrorMsg())

}

func (suite *ReportServiceTestSuite) TestGetReportHistory() {
	suite.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		suite.T().Skip("Skip test when reporting enhancements are disabled")
		suite.T().SkipNow()
	}
	ctx := context.Background()
	reportSnapshot := &storage.ReportSnapshot{
		ReportId: "test_report",
		Name:     "Report",
		ReportStatus: &storage.ReportStatus{
			ErrorMsg:                 "Error msg",
			ReportNotificationMethod: 1,
		},
		ReportConfigurationId: "test_report",
	}

	suite.snapshotDS.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	s := serviceImpl{snapshotDatastore: suite.snapshotDS}
	emptyQuery := &apiV2.RawQuery{Query: ""}
	req := &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	res, err := s.GetReportHistory(ctx, req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetId(), "test_report")
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "",
		ReportParamQuery: emptyQuery,
	}

	_, err = s.GetReportHistory(ctx, req)
	assert.Error(suite.T(), err)

	query := &apiV2.RawQuery{Query: "Report Name:test_report"}
	req = &apiV2.GetReportHistoryRequest{
		ReportConfigId:   "test_report_config",
		ReportParamQuery: query,
	}

	res, err = s.GetReportHistory(ctx, req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetId(), "test_report")
	assert.Equal(suite.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")

}

func (suite *ReportServiceTestSuite) TestAuthz() {

	suite.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		suite.T().Skip("Skip test when reporting enhancements are disabled")
		suite.T().SkipNow()
	}

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
