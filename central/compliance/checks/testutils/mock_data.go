package testutils

import (
	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/central/compliance/framework/mocks"
)

// SetupMockCtxAndMockData sets up a mock compliance context, and a mock data repository.
// It also returns a pointer to a slice that will contain all evidence recorded through the context.
// Callers in tests can use mockData to inject whatever mock data they want.
func SetupMockCtxAndMockData(ctrl *gomock.Controller) (*mocks.MockComplianceContext, *mocks.MockComplianceDataRepository, *EvidenceRecords) {
	var records EvidenceRecords
	mockCtx := mocks.NewMockComplianceContext(ctrl)
	mockData := mocks.NewMockComplianceDataRepository(ctrl)
	mockCtx.EXPECT().Data().AnyTimes().Return(mockData)
	mockCtx.EXPECT().RecordEvidence(gomock.Any(), gomock.Any()).AnyTimes().Do(func(status framework.Status, message string) {
		records.List = append(records.List, framework.EvidenceRecord{Status: status, Message: message})
	})
	return mockCtx, mockData, &records
}
