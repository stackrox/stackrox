package fixtures

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// GetReportSnapshot returns a valid report snapshot object
func GetReportSnapshot() *storage.ReportSnapshot {
	return &storage.ReportSnapshot{
		ReportConfigurationId: "config-1",
		Name:                  "App Team 1 Report",
		Description:           "Report for CVEs in app team 1's infrastructure",
		Type:                  storage.ReportSnapshot_VULNERABILITY,
		Collection: &storage.CollectionSnapshot{
			Id:   "collection-1",
			Name: "collection-1",
		},
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_WEEKLY,
			Interval: &storage.Schedule_DaysOfWeek_{
				DaysOfWeek: &storage.Schedule_DaysOfWeek{
					Days: []int32{1},
				},
			},
		},
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_PREPARING,
			QueuedAt:                 timestamp.TimestampNow(),
			CompletedAt:              timestamp.TimestampNow(),
			ErrorMsg:                 "",
			ReportRequestType:        storage.ReportStatus_ON_DEMAND,
			ReportNotificationMethod: storage.ReportStatus_EMAIL,
		},
		Notifiers: []*storage.NotifierSnapshot{
			{
				NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "email-notifier-yahoo",
						MailingLists: []string{"foo@yahoo.com"},
					},
				},
				NotifierName: "email-notifier-yahoo",
			},
			{
				NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "email-notifier-gmail",
						MailingLists: []string{"bar@gmail.com"},
					},
				},
				NotifierName: "email-notifier-gmail",
			},
		},
		Requester: &storage.SlimUser{
			Id:   "user-1",
			Name: "user-1",
		},
	}
}
