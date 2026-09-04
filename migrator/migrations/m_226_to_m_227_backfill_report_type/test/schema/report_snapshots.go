// Frozen pre-migration schema for the report_snapshots table (without the type column).

package schema

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableReportSnapshotsStmt holds the create statement for table `report_snapshots` (pre-migration).
	CreateTableReportSnapshotsStmt = &postgres.CreateStmts{
		GormModel: (*ReportSnapshots)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// ReportSnapshotsTableName specifies the name of the table in postgres.
	ReportSnapshotsTableName = "report_snapshots"
)

// ReportSnapshots holds the pre-migration Gorm model (no type column).
type ReportSnapshots struct {
	ReportID                             string                                  `gorm:"column:reportid;type:uuid;primaryKey"`
	ReportConfigurationID                string                                  `gorm:"column:reportconfigurationid;type:varchar"`
	Name                                 string                                  `gorm:"column:name;type:varchar"`
	ReportStatusRunState                 storage.ReportStatus_RunState           `gorm:"column:reportstatus_runstate;type:integer"`
	ReportStatusQueuedAt                 *time.Time                              `gorm:"column:reportstatus_queuedat;type:timestamp"`
	ReportStatusCompletedAt              *time.Time                              `gorm:"column:reportstatus_completedat;type:timestamp"`
	ReportStatusReportRequestType        storage.ReportStatus_RunMethod          `gorm:"column:reportstatus_reportrequesttype;type:integer"`
	ReportStatusReportNotificationMethod storage.ReportStatus_NotificationMethod `gorm:"column:reportstatus_reportnotificationmethod;type:integer"`
	RequesterID                          string                                  `gorm:"column:requester_id;type:varchar"`
	RequesterName                        string                                  `gorm:"column:requester_name;type:varchar"`
	AreaOfConcern                        string                                  `gorm:"column:areaofconcern;type:varchar"`
	Serialized                           []byte                                  `gorm:"column:serialized;type:bytea"`
}
