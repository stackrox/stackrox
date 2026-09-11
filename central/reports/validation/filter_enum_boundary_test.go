package validation

import (
	"testing"

	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateAndGenerateReportRequestRejectsInvalidNotificationMethodBeforeLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validator := &Validator{reportConfigDatastore: reportConfigDSMocks.NewMockDataStore(ctrl)}

	_, err := validator.ValidateAndGenerateReportRequest(
		"config",
		storage.ReportStatus_NotificationMethod(99),
		storage.ReportStatus_ON_DEMAND,
		nil,
	)
	require.ErrorIs(t, err, errox.InvalidArgs)
}

func TestValidateImageFiltersRejectsInvalidEnumsAndDates(t *testing.T) {
	tests := map[string]*apiV2.VulnerabilityReportFilters{
		"image type": {
			ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{apiV2.VulnerabilityReportFilters_ImageType(99)},
			CvesSince:  &apiV2.VulnerabilityReportFilters_AllVuln{AllVuln: true},
		},
		"fixability": {
			ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{apiV2.VulnerabilityReportFilters_DEPLOYED},
			Fixability:  apiV2.VulnerabilityReportFilters_Fixability(99),
			CvesSince:   &apiV2.VulnerabilityReportFilters_AllVuln{AllVuln: true},
		},
		"severity": {
			ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{apiV2.VulnerabilityReportFilters_DEPLOYED},
			Severities: []apiV2.VulnerabilityReportFilters_VulnerabilitySeverity{apiV2.VulnerabilityReportFilters_VulnerabilitySeverity(99)},
			CvesSince:  &apiV2.VulnerabilityReportFilters_AllVuln{AllVuln: true},
		},
		"nil start date": {
			ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{apiV2.VulnerabilityReportFilters_DEPLOYED},
			CvesSince:  &apiV2.VulnerabilityReportFilters_SinceStartDate{},
		},
	}

	for name, filters := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, (&Validator{}).validateImageFilters(filters), errox.InvalidArgs)
		})
	}
}
