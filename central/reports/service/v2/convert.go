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
		ReportNotificationMethod: apiV2.ReportStatus_NotificationMethod(status.GetReportNotificationMethod()),
		ErrorMsg:                 status.GetErrorMsg(),
	}

}

// ConvertProtoNotifierSnapshotToV2 converts storage.NotifierSnapshot to apiV2.NotifierConfiguration
func ConvertProtoNotifierSnapshotToV2(notifierConfig *storage.NotifierSnapshot) (*apiV2.NotifierConfiguration, error) {
	if notifierConfig == nil {
		return nil, nil
	}

	ret := &apiV2.NotifierConfiguration{}
	if notifierConfig.GetEmailConfig() != nil {
		emailConfig := &apiV2.EmailNotifierConfiguration{}

		emailConfig.MailingLists = append(emailConfig.MailingLists, notifierConfig.GetEmailConfig().GetMailingLists()...)

		ret.NotifierConfig = &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: emailConfig,
		}

		notifierName := notifierConfig.GetNotifierName()
		ret.NotifierName = notifierName
	}
	return ret, nil
}

func convertprotoV2ReportCollection(collection *storage.CollectionSnapshot) *apiV2.CollectionSnapshot {
	if collection == nil {
		return nil
	}

	return &apiV2.CollectionSnapshot{
		Id:   collection.GetId(),
		Name: collection.GetName(),
	}
}

// convertPrototoV2ReportSnapshot converts storage.ReportSnapshot to apiV2.ReportSnapshot
func convertPrototoV2ReportSnapshot(snapshots []*storage.ReportSnapshot) []*apiV2.ReportSnapshot {
	if snapshots == nil {
		return nil
	}
	res := []*apiV2.ReportSnapshot{}
	for _, snapshot := range snapshots {
		snapshotv2 := &apiV2.ReportSnapshot{
			ReportStatus:       convertPrototoV2Reportstatus(snapshot.GetReportStatus()),
			Id:                 snapshot.GetReportId(),
			Name:               snapshot.GetName(),
			Description:        snapshot.GetDescription(),
			CollectionSnapshot: convertprotoV2ReportCollection(snapshot.GetCollection()),
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
			converted, err := ConvertProtoNotifierSnapshotToV2(notifier)
			if err != nil {
				return nil
			}
			snapshotv2.Notifiers = append(snapshotv2.Notifiers, converted)
		}
		res = append(res, snapshotv2)

	}

	return res

}
