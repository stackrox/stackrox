package service

import (
	"context"
	"testing"

	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	managerMocks "github.com/stackrox/rox/central/reports/manager/mocks"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestConfigSeparation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	reportConfigStore := reportConfigDSMocks.NewMockDataStore(mockCtrl)
	manager := managerMocks.NewMockManager(mockCtrl)
	service := New(reportConfigStore, nil, manager)
	ctx := context.Background()

	// Error on v2 config
	configV2 := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), configV2.Id).Return(configV2, true, nil).Times(1)
	_, err := service.RunReport(ctx, &apiV1.ResourceByID{Id: configV2.Id})
	assert.Error(t, err)

	// No error on v1 config
	configV1 := fixtures.GetValidReportConfigWithMultipleNotifiersV1()
	reportConfigStore.EXPECT().GetReportConfiguration(gomock.Any(), configV1.Id).Return(configV1, true, nil).Times(1)
	manager.EXPECT().RunReport(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	_, err = service.RunReport(ctx, &apiV1.ResourceByID{Id: configV1.Id})
	assert.NoError(t, err)
}