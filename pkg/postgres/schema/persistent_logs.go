package schema

import (
	"reflect"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTablePersistentLogsStmt holds the create statement for table `persistent_logs`.
	// Serial log_id allows for inserts and no updates to speed up writes dramatically
	CreateTablePersistentLogsStmt = &postgres.CreateStmts{
		GormModel: (*NetworkFlows)(nil),
	}

	// PersistentLogsSchema is the go schema for table `persistent_logs`.
	PersistentLogsSchema = func() *walker.Schema {
		schema := GetSchemaForTable("persistent_logs")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.PersistentLog)(nil)), "persistent_logs")
		RegisterTable(schema, CreateTablePersistentLogsStmt)
		return schema
	}()
)

const (
	// PersistentLogsTableName holds the database table name
	PersistentLogsTableName = "persistent_logs"
)

// PersistentLogs holds the Gorm model for Postgres table `network_flows`.
type PersistentLogs struct {
	LogID     string     `gorm:"column:log_id;type:bigserial;primaryKey"`
	Log       string     `gorm:"column:log;type:text"`
	Timestamp *time.Time `gorm:"column:timestamp;type:timestamp;index:persistent_logs_timestamp,type:btree"`
}
