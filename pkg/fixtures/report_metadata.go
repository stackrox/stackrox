package fixtures

import (
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

func GetReportMetadata() *storage.ReportMetadata {
	return &storage.ReportMetadata{
		ReportId:       uuid.NewV4().String(),
		ReportConfigId: "config-1",
		User: &storage.SlimUser{
			Id:   "user-1",
			Name: "user-1",
		},
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_SUCCESS,
			QueuedAt:                 timestamp.TimestampNow(),
			CompletedAt:              timestamp.TimestampNow(),
			ErrorMsg:                 "",
			ReportNotificationMethod: storage.ReportStatus_EMAIL,
		},
		IsDownloaded: false,
	}
}
