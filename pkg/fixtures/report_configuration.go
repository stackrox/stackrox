package fixtures

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
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

// GetValidReportConfigWithMultipleNotifiersV1 returns a valid storage report configuration object with 2 email notifier configs for v1 workflow
func GetValidReportConfigWithMultipleNotifiersV1() *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		Filter: &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{
				Fixability: storage.VulnerabilityReportFilters_FIXABLE,
				Severities: []storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
				ImageTypes: []storage.VulnerabilityReportFilters_ImageType{
					storage.VulnerabilityReportFilters_DEPLOYED,
					storage.VulnerabilityReportFilters_WATCHED,
				},
				CvesSince: &storage.VulnerabilityReportFilters_SinceLastSentScheduledReport{
					SinceLastSentScheduledReport: true,
				},
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
		ScopeId: "collection-1",
		NotifierConfig: &storage.ReportConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				NotifierId:   "email-notifier-yahoo",
				MailingLists: []string{"foo@yahoo.com"},
			},
		},
	}
}

// GetValidReportConfigWithMultipleNotifiersV2 returns a valid storage report configuration object with 2 email notifier configs for v2 workflow
func GetValidReportConfigWithMultipleNotifiersV2() *storage.ReportConfiguration {
	return &storage.ReportConfiguration{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		Version:     2,
		Filter: &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{
				Fixability: storage.VulnerabilityReportFilters_FIXABLE,
				Severities: []storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
				ImageTypes: []storage.VulnerabilityReportFilters_ImageType{
					storage.VulnerabilityReportFilters_DEPLOYED,
					storage.VulnerabilityReportFilters_WATCHED,
				},
				CvesSince: &storage.VulnerabilityReportFilters_SinceLastSentScheduledReport{
					SinceLastSentScheduledReport: true,
				},
			},
		},
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_WEEKLY,
			Interval: &storage.Schedule_DaysOfWeek_{
				DaysOfWeek: &storage.Schedule_DaysOfWeek{
					Days: []int32{2},
				},
			},
		},
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{
				CollectionId: "collection-1",
			},
		},
		Notifiers: []*storage.NotifierConfiguration{
			{
				Ref: &storage.NotifierConfiguration_Id{
					Id: "email-notifier-yahoo",
				},
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						MailingLists: []string{"foo@yahoo.com"},
					},
				},
			},
			{
				Ref: &storage.NotifierConfiguration_Id{
					Id: "email-notifier-gmail",
				},
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						MailingLists: []string{"bar@gmail.com"},
					},
				},
			},
		},
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

// GetInvalidReportConfigurationMissingDaysOfWeek returns a mock report configuration with an invalid schedule that is
// missing days of week
func GetInvalidReportConfigurationMissingDaysOfWeek() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.Schedule = &storage.Schedule{
		IntervalType: storage.Schedule_WEEKLY,
		Interval: &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{
				Days: []int32{},
			},
		},
	}
	return rc
}

// GetInvalidReportConfigurationMissingDaysOfMonth returns a mock report configuration with an invalid schedule that is
// missing days of month
func GetInvalidReportConfigurationMissingDaysOfMonth() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.Schedule = &storage.Schedule{
		IntervalType: storage.Schedule_MONTHLY,
		Interval: &storage.Schedule_DaysOfMonth_{
			DaysOfMonth: &storage.Schedule_DaysOfMonth{
				Days: nil,
			},
		},
	}
	return rc
}

// GetInvalidReportConfigurationDailySchedule returns a mock report configuration with daily intervalType in schedule
func GetInvalidReportConfigurationDailySchedule() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.Schedule = &storage.Schedule{
		IntervalType: storage.Schedule_DAILY,
		Interval:     nil,
	}
	return rc
}

// GetInvalidReportConfigurationIncorrectEmailV1 returns a mock report configuration with incorrect email
func GetInvalidReportConfigurationIncorrectEmailV1() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()

	rc.NotifierConfig = &storage.ReportConfiguration_EmailConfig{
		EmailConfig: &storage.EmailNotifierConfiguration{
			NotifierId:   "email-notifier-gmail",
			MailingLists: []string{"sdfdksfjk"},
		},
	}
	return rc
}

// GetValidV2ReportConfigWithMultipleNotifiers returns a valid v2 api report configuration object with 2 email notifier configs
func GetValidV2ReportConfigWithMultipleNotifiers() *v2.ReportConfiguration {
	return &v2.ReportConfiguration{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        v2.ReportConfiguration_VULNERABILITY,
		Filter: &v2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &v2.VulnerabilityReportFilters{
				Fixability: v2.VulnerabilityReportFilters_FIXABLE,
				Severities: []v2.VulnerabilityReportFilters_VulnerabilitySeverity{v2.VulnerabilityReportFilters_CRITICAL_VULNERABILITY_SEVERITY},
				ImageTypes: []v2.VulnerabilityReportFilters_ImageType{
					v2.VulnerabilityReportFilters_DEPLOYED,
					v2.VulnerabilityReportFilters_WATCHED,
				},
				CvesSince: &v2.VulnerabilityReportFilters_SinceLastSentScheduledReport{
					SinceLastSentScheduledReport: true,
				},
			},
		},
		Schedule: &v2.ReportSchedule{
			IntervalType: v2.ReportSchedule_WEEKLY,
			Interval: &v2.ReportSchedule_DaysOfWeek_{
				DaysOfWeek: &v2.ReportSchedule_DaysOfWeek{
					Days: []int32{2},
				},
			},
		},
		ResourceScope: &v2.ResourceScope{
			ScopeReference: &v2.ResourceScope_CollectionScope{
				CollectionScope: &v2.CollectionReference{
					CollectionId:   "collection-1",
					CollectionName: "collection-1",
				},
			},
		},
		Notifiers: []*v2.NotifierConfiguration{
			{
				NotifierConfig: &v2.NotifierConfiguration_EmailConfig{
					EmailConfig: &v2.EmailNotifierConfiguration{
						NotifierId:   "email-notifier-yahoo",
						MailingLists: []string{"foo@yahoo.com"},
					},
				},
				NotifierName: "email-notifier-yahoo",
			},
			{
				NotifierConfig: &v2.NotifierConfiguration_EmailConfig{
					EmailConfig: &v2.EmailNotifierConfiguration{
						NotifierId:   "email-notifier-gmail",
						MailingLists: []string{"bar@gmail.com"},
					},
				},
				NotifierName: "email-notifier-gmail",
			},
		},
	}
}
