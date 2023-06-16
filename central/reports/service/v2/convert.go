package v2

import (
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

func convertPrototoV2Reportstatus(status *storage.ReportStatus) *apiV2.ReportStatus {
	ret := &apiV2.ReportStatus{}
	if status == nil {
		return ret
	}

	ret.ReportMethod = apiV2.ReportStatus_ReportMethod(status.GetReportRequestType())
	ret.RunTime = status.GetCompletedAt()
	ret.RunState = apiV2.ReportStatus_RunState(status.GetRunState())
	ret.ReportNotificationMethod = apiV2.ReportStatus_NotificationMethod(status.GetReportNotificationMethod())
	ret.ErrorMsg = status.GetErrorMsg()
	return ret

}
