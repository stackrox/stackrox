// Frozen copy of the generated schema for table `sensor_upgrade_configs`, taken at
// the time the table was removed (sensor auto-upgrader removal). Kept here so this
// migration can reference the table name and create statement without depending on
// the live schema package, which no longer defines them. DO NOT EDIT.

package old

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableSensorUpgradeConfigsStmt holds the create statement for table `sensor_upgrade_configs`.
	CreateTableSensorUpgradeConfigsStmt = &postgres.CreateStmts{
		GormModel: (*SensorUpgradeConfigs)(nil),
		Children:  []*postgres.CreateStmts{},
		PostStmts: []string{
			"ALTER TABLE sensor_upgrade_configs REPLICA IDENTITY FULL",
		},
	}
)

const (
	// SensorUpgradeConfigsTableName specifies the name of the table in postgres.
	SensorUpgradeConfigsTableName = "sensor_upgrade_configs"
)

// SensorUpgradeConfigs holds the Gorm model for Postgres table `sensor_upgrade_configs`.
type SensorUpgradeConfigs struct {
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
