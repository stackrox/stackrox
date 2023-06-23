package v2

import (
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
