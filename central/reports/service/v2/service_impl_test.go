package v2

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	metadataDSMocks "github.com/stackrox/rox/central/reports/metadata/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestCollectionService(t *testing.T) {
	suite.Run(t, new(ReportServiceTestSuite))
}

type ReportServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	reportDS          *metadataDSMocks.MockDataStore
	collectionService Service
}

func (suite *ReportServiceTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.reportDS = metadataDSMocks.NewMockDataStore(suite.mockCtrl)

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
	s := serviceImpl{metadataDatastore: suite.reportDS}
	repStatus, err := s.GetReportStatus(ctx, &id)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), repStatus.GetErrorMsg(), status.GetErrorMsg())

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
