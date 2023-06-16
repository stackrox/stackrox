package v2

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	metadataDSMocks "github.com/stackrox/rox/central/reports/metadata/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
)

func TestGetReportStatus(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	t.Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		t.Skip("Skip test when reporting enhancements are disabled")
		t.SkipNow()
	}
	ctx := context.Background()

	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	metadata := &storage.ReportMetadata{
		ReportId:     "test_report",
		ReportStatus: status,
	}
	metadataDS := metadataDSMocks.NewMockDataStore(mockCtrl)
	metadataDS.EXPECT().Get(gomock.Any(), gomock.Any()).Return(metadata, true, nil)
	id := apiV2.ResourceByID{
		Id: "test_report",
	}
	s := serviceImpl{metadataDatastore: metadataDS}
	repStatus, err := s.GetReportStatus(ctx, &id)
	assert.NoError(t, err)
	assert.Equal(t, repStatus.GetErrorMsg(), status.GetErrorMsg())

}
