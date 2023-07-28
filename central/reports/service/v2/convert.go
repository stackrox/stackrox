package v2

import (
	reportConfig "github.com/stackrox/rox/central/reportconfigurations/service/v2"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func convertPrototoV2Reportstatus(status *storage.ReportStatus) *apiV2.ReportStatus {
	if status == nil {
		return nil
	}
	return &apiV2.ReportStatus{
		ReportRequestType:        apiV2.ReportStatus_ReportMethod(status.GetReportRequestType()),
		CompletedAt:              status.GetCompletedAt(),
		RunState:                 apiV2.ReportStatus_RunState(status.GetRunState()),
		ReportNotificationMethod: apiV2.NotificationMethod(status.GetReportNotificationMethod()),
		ErrorMsg:                 status.GetErrorMsg(),
	}

}

func convertProtoReportCollectiontoV2(collection *storage.CollectionSnapshot) *apiV2.CollectionSnapshot {
	if collection == nil {
		return nil
	}

	return &apiV2.CollectionSnapshot{
		Id:   collection.GetId(),
		Name: collection.GetName(),
	}
}

// ConvertProtoNotifierSnapshotToV2 converts notifiersnapshot proto to v2
func ConvertProtoNotifierSnapshotToV2(notifierSnapshot *storage.NotifierSnapshot) *apiV2.NotifierSnapshot {
	if notifierSnapshot == nil {
		return nil
	}
	if notifierSnapshot.GetEmailConfig() == nil {
		return &apiV2.NotifierSnapshot{}
	}

	return &apiV2.NotifierSnapshot{
		NotifierName: notifierSnapshot.GetNotifierName(),
		NotifierConfig: &apiV2.NotifierSnapshot_EmailConfig{
			EmailConfig: &apiV2.EmailNotifierConfiguration{
				MailingLists: notifierSnapshot.GetEmailConfig().GetMailingLists(),
			},
		},
	}
}

// convertPrototoV2ReportSnapshot converts storage.ReportSnapshot to apiV2.ReportSnapshot
func convertProtoReportSnapshotstoV2(snapshots []*storage.ReportSnapshot) []*apiV2.ReportSnapshot {
	if snapshots == nil {
		return nil
	}
	res := make([]*apiV2.ReportSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotv2 := &apiV2.ReportSnapshot{
			ReportStatus:       convertPrototoV2Reportstatus(snapshot.GetReportStatus()),
			Id:                 snapshot.GetReportConfigurationId(),
			Name:               snapshot.GetName(),
			Description:        snapshot.GetDescription(),
			CollectionSnapshot: convertProtoReportCollectiontoV2(snapshot.GetCollection()),
			User: &apiV2.SlimUser{
				Id:   snapshot.GetRequester().GetId(),
				Name: snapshot.GetRequester().GetId(),
			},
			Schedule: reportConfig.ConvertProtoScheduleToV2(snapshot.GetSchedule()),
			Filter: &apiV2.ReportSnapshot_VulnReportFilters{
				VulnReportFilters: reportConfig.ConvertProtoVulnReportFiltersToV2(snapshot.GetVulnReportFilters()),
			},
		}
		for _, notifier := range snapshot.GetNotifiers() {
			converted := ConvertProtoNotifierSnapshotToV2(notifier)
			if converted != nil {
				snapshotv2.Notifiers = append(snapshotv2.Notifiers, converted)
			}
		}
		res = append(res, snapshotv2)

	}

	return res

}
