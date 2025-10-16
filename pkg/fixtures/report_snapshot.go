package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/proto"
)

// GetReportSnapshot returns a valid report snapshot object
func GetReportSnapshot() *storage.ReportSnapshot {
	return storage.ReportSnapshot_builder{
		ReportConfigurationId: "config-1",
		Name:                  "App Team 1 Report",
		Description:           "Report for CVEs in app team 1's infrastructure",
		Type:                  storage.ReportSnapshot_VULNERABILITY,
		Collection: storage.CollectionSnapshot_builder{
			Id:   "collection-1",
			Name: "collection-1",
		}.Build(),
		Schedule: storage.Schedule_builder{
			IntervalType: storage.Schedule_WEEKLY,
			DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
				Days: []int32{1},
			}.Build(),
		}.Build(),
		VulnReportFilters: storage.VulnerabilityReportFilters_builder{
			Fixability: storage.VulnerabilityReportFilters_BOTH,
			Severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
			ImageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
				storage.VulnerabilityReportFilters_WATCHED,
			},
			AllVuln: proto.Bool(true),
		}.Build(),
		ReportStatus: storage.ReportStatus_builder{
			RunState:                 storage.ReportStatus_PREPARING,
			QueuedAt:                 protocompat.TimestampNow(),
			CompletedAt:              protocompat.TimestampNow(),
			ErrorMsg:                 "",
			ReportRequestType:        storage.ReportStatus_ON_DEMAND,
			ReportNotificationMethod: storage.ReportStatus_EMAIL,
		}.Build(),
		Notifiers: []*storage.NotifierSnapshot{
			storage.NotifierSnapshot_builder{
				EmailConfig: storage.EmailNotifierConfiguration_builder{
					NotifierId:   "email-notifier-yahoo",
					MailingLists: []string{"foo@yahoo.com"},
				}.Build(),
				NotifierName: "email-notifier-yahoo",
			}.Build(),
			storage.NotifierSnapshot_builder{
				EmailConfig: storage.EmailNotifierConfiguration_builder{
					NotifierId:   "email-notifier-gmail",
					MailingLists: []string{"bar@gmail.com"},
				}.Build(),
				NotifierName: "email-notifier-gmail",
			}.Build(),
		},
		Requester: storage.SlimUser_builder{
			Id:   "user-1",
			Name: "user-1",
		}.Build(),
	}.Build()
}
