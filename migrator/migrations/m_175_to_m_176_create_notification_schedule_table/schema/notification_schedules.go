package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableNotificationSchedulesStmt holds the create statement for table `notification_schedules`.
	CreateTableNotificationSchedulesStmt = &postgres.CreateStmts{
		GormModel: (*NotificationSchedules)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// NotificationSchedulesSchema is the go schema for table `notification_schedules`.
	NotificationSchedulesSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.NotificationSchedule)(nil)), "notification_schedules")
		return schema
	}()
)

// NotificationSchedules holds the Gorm model for Postgres table `notification_schedules`.
type NotificationSchedules struct {
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
