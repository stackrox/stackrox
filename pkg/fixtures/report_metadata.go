package fixtures

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetReportMetadata returns a valid report metadata object
func GetReportMetadata() *storage.ReportMetadata {
	return &storage.ReportMetadata{
		ReportId:       uuid.NewV4().String(),
		ReportConfigId: "config-1",
		Requester: &storage.SlimUser{
			Id:   "user-1",
			Name: "user-1",
		},
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_SUCCESS,
			QueuedAt:                 timestamp.TimestampNow(),
			CompletedAt:              timestamp.TimestampNow(),
			ErrorMsg:                 "",
			ReportMethod:             storage.ReportStatus_ON_DEMAND,
			ReportNotificationMethod: storage.ReportStatus_EMAIL,
		},
		IsDownloaded: false,
	}
}
