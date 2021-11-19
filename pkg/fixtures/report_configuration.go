package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetValidReportConfiguration returns a mock report configuration
func GetValidReportConfiguration() *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		Filter: &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{
				Fixability:      storage.VulnerabilityReportFilters_FIXABLE,
				SinceLastReport: false,
				Severities:      []storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
			},
		},
		ScopeId: "scope-1",
		NotifierConfig: &storage.ReportConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				NotifierId:   "email-notifier-gmail",
				MailingLists: []string{"foo@yahoo.com"},
			},
		},
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_WEEKLY,
			Interval: &storage.Schedule_DaysOfWeek_{
				DaysOfWeek: &storage.Schedule_DaysOfWeek{
					Days: []int32{1},
				},
			},
		},
		LastRunStatus:         nil,
		LastSuccessfulRunTime: nil,
	}
}

// GetInvalidReportConfigurationNoNotifier returns a mock report configuration without a notifier
func GetInvalidReportConfigurationNoNotifier() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.NotifierConfig = nil
	return rc
}

// GetInvalidReportConfigurationIncorrectSchedule returns a mock report configuration with an invalid schedule
func GetInvalidReportConfigurationIncorrectSchedule() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.Schedule = &storage.Schedule{
		IntervalType: storage.Schedule_WEEKLY,
		Interval: &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{
				Days: []int32{8},
			},
		},
	}
	return rc
}

// GetInvalidReportConfigurationMissingSchedule returns a mock report configuration without a schedule
func GetInvalidReportConfigurationMissingSchedule() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.Schedule = nil
	return rc
}

// GetInvalidReportConfigurationIncorrectEmail returns a mock report configuration with incorrect email
func GetInvalidReportConfigurationIncorrectEmail() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()

	rc.NotifierConfig = &storage.ReportConfiguration_EmailConfig{
		EmailConfig: &storage.EmailNotifierConfiguration{
			NotifierId:   "email-notifier-gmail",
			MailingLists: []string{"sdfdksfjk"},
		},
	}
	return rc
}
