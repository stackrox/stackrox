package fixtures

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
)

// GetValidReportConfiguration returns a mock report configuration
func GetValidReportConfiguration() *storage.ReportConfiguration {
	return storage.ReportConfiguration_builder{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		VulnReportFilters: storage.VulnerabilityReportFilters_builder{
			Fixability:      storage.VulnerabilityReportFilters_FIXABLE,
			SinceLastReport: false,
			Severities:      []storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
		}.Build(),
		ScopeId: "scope-1",
		EmailConfig: storage.EmailNotifierConfiguration_builder{
			NotifierId:   "email-notifier-gmail",
			MailingLists: []string{"foo@yahoo.com"},
		}.Build(),
		Schedule: storage.Schedule_builder{
			IntervalType: storage.Schedule_WEEKLY,
			DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
				Days: []int32{1},
			}.Build(),
		}.Build(),
		LastRunStatus:         nil,
		LastSuccessfulRunTime: nil,
	}.Build()
}

// GetValidReportConfigWithMultipleNotifiersV1 returns a valid storage report configuration object with 2 email notifier configs for v1 workflow
func GetValidReportConfigWithMultipleNotifiersV1() *storage.ReportConfiguration {
	return storage.ReportConfiguration_builder{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		VulnReportFilters: storage.VulnerabilityReportFilters_builder{
			Fixability: storage.VulnerabilityReportFilters_FIXABLE,
			Severities: []storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
			ImageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
				storage.VulnerabilityReportFilters_WATCHED,
			},
			SinceLastSentScheduledReport: proto.Bool(true),
		}.Build(),
		Schedule: storage.Schedule_builder{
			IntervalType: storage.Schedule_WEEKLY,
			DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
				Days: []int32{1},
			}.Build(),
		}.Build(),
		ScopeId: "collection-1",
		EmailConfig: storage.EmailNotifierConfiguration_builder{
			NotifierId:   "email-notifier-yahoo",
			MailingLists: []string{"foo@yahoo.com"},
		}.Build(),
	}.Build()
}

// GetValidReportConfigWithMultipleNotifiersV2 returns a valid storage report configuration object with 2 email notifier configs for v2 workflow
func GetValidReportConfigWithMultipleNotifiersV2() *storage.ReportConfiguration {
	return storage.ReportConfiguration_builder{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        storage.ReportConfiguration_VULNERABILITY,
		Version:     2,
		VulnReportFilters: storage.VulnerabilityReportFilters_builder{
			Fixability: storage.VulnerabilityReportFilters_FIXABLE,
			Severities: []storage.VulnerabilitySeverity{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY},
			ImageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
				storage.VulnerabilityReportFilters_WATCHED,
			},
			SinceLastSentScheduledReport: proto.Bool(true),
		}.Build(),
		Schedule: storage.Schedule_builder{
			IntervalType: storage.Schedule_WEEKLY,
			DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
				Days: []int32{2},
			}.Build(),
		}.Build(),
		ResourceScope: storage.ResourceScope_builder{
			CollectionId: proto.String("collection-1"),
		}.Build(),
		Notifiers: []*storage.NotifierConfiguration{
			storage.NotifierConfiguration_builder{
				Id: proto.String("email-notifier-yahoo"),
				EmailConfig: storage.EmailNotifierConfiguration_builder{
					MailingLists: []string{"foo@yahoo.com"},
				}.Build(),
			}.Build(),
			storage.NotifierConfiguration_builder{
				Id: proto.String("email-notifier-gmail"),
				EmailConfig: storage.EmailNotifierConfiguration_builder{
					MailingLists: []string{"bar@gmail.com"},
				}.Build(),
			}.Build(),
		},
	}.Build()
}

// GetInvalidReportConfigurationNoNotifier returns a mock report configuration without a notifier
func GetInvalidReportConfigurationNoNotifier() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.ClearNotifierConfig()
	return rc
}

// GetInvalidReportConfigurationIncorrectSchedule returns a mock report configuration with an invalid schedule
func GetInvalidReportConfigurationIncorrectSchedule() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.SetSchedule(storage.Schedule_builder{
		IntervalType: storage.Schedule_WEEKLY,
		DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
			Days: []int32{8},
		}.Build(),
	}.Build())
	return rc
}

// GetInvalidReportConfigurationMissingSchedule returns a mock report configuration without a schedule
func GetInvalidReportConfigurationMissingSchedule() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.ClearSchedule()
	return rc
}

// GetInvalidReportConfigurationMissingDaysOfWeek returns a mock report configuration with an invalid schedule that is
// missing days of week
func GetInvalidReportConfigurationMissingDaysOfWeek() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	rc.SetSchedule(storage.Schedule_builder{
		IntervalType: storage.Schedule_WEEKLY,
		DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
			Days: []int32{},
		}.Build(),
	}.Build())
	return rc
}

// GetInvalidReportConfigurationMissingDaysOfMonth returns a mock report configuration with an invalid schedule that is
// missing days of month
func GetInvalidReportConfigurationMissingDaysOfMonth() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	sd := &storage.Schedule_DaysOfMonth{}
	sd.SetDays(nil)
	schedule := &storage.Schedule{}
	schedule.SetIntervalType(storage.Schedule_MONTHLY)
	schedule.SetDaysOfMonth(proto.ValueOrDefault(sd))
	rc.SetSchedule(schedule)
	return rc
}

// GetInvalidReportConfigurationDailySchedule returns a mock report configuration with daily intervalType in schedule
func GetInvalidReportConfigurationDailySchedule() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()
	schedule := &storage.Schedule{}
	schedule.SetIntervalType(storage.Schedule_DAILY)
	schedule.ClearInterval()
	rc.SetSchedule(schedule)
	return rc
}

// GetInvalidReportConfigurationIncorrectEmailV1 returns a mock report configuration with incorrect email
func GetInvalidReportConfigurationIncorrectEmailV1() *storage.ReportConfiguration {
	rc := GetValidReportConfiguration()

	enc := &storage.EmailNotifierConfiguration{}
	enc.SetNotifierId("email-notifier-gmail")
	enc.SetMailingLists([]string{"sdfdksfjk"})
	rc.SetEmailConfig(proto.ValueOrDefault(enc))
	return rc
}

// GetValidV2ReportConfigWithMultipleNotifiers returns a valid v2 api report configuration object with 2 email notifier configs
func GetValidV2ReportConfigWithMultipleNotifiers() *v2.ReportConfiguration {
	return v2.ReportConfiguration_builder{
		Id:          "report1",
		Name:        "App Team 1 Report",
		Description: "Report for CVEs in app team 1's infrastructure",
		Type:        v2.ReportConfiguration_VULNERABILITY,
		VulnReportFilters: v2.VulnerabilityReportFilters_builder{
			Fixability: v2.VulnerabilityReportFilters_FIXABLE,
			Severities: []v2.VulnerabilityReportFilters_VulnerabilitySeverity{v2.VulnerabilityReportFilters_CRITICAL_VULNERABILITY_SEVERITY},
			ImageTypes: []v2.VulnerabilityReportFilters_ImageType{
				v2.VulnerabilityReportFilters_DEPLOYED,
				v2.VulnerabilityReportFilters_WATCHED,
			},
			SinceLastSentScheduledReport: proto.Bool(true),
		}.Build(),
		Schedule: v2.ReportSchedule_builder{
			IntervalType: v2.ReportSchedule_WEEKLY,
			DaysOfWeek: v2.ReportSchedule_DaysOfWeek_builder{
				Days: []int32{2},
			}.Build(),
		}.Build(),
		ResourceScope: v2.ResourceScope_builder{
			CollectionScope: v2.CollectionReference_builder{
				CollectionId:   "collection-1",
				CollectionName: "collection-1",
			}.Build(),
		}.Build(),
		Notifiers: []*v2.NotifierConfiguration{
			v2.NotifierConfiguration_builder{
				EmailConfig: v2.EmailNotifierConfiguration_builder{
					NotifierId:   "email-notifier-yahoo",
					MailingLists: []string{"foo@yahoo.com"},
				}.Build(),
				NotifierName: "email-notifier-yahoo",
			}.Build(),
			v2.NotifierConfiguration_builder{
				EmailConfig: v2.EmailNotifierConfiguration_builder{
					NotifierId:   "email-notifier-gmail",
					MailingLists: []string{"bar@gmail.com"},
				}.Build(),
				NotifierName: "email-notifier-gmail",
			}.Build(),
		},
	}.Build()
}
