package m227tom228

import (
	"fmt"

	"github.com/pkg/errors"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_227_to_m_228_remove_sensor_upgrade_configs_table/schema/old"
	"github.com/stackrox/rox/migrator/types"
)

const dropTable = "DROP TABLE IF EXISTS %s"

// migrate drops the orphaned sensor_upgrade_configs table. The sensor auto-upgrader
// has been removed, so nothing reads or writes this table anymore. GORM AutoMigrate
// never drops tables, hence this explicit DDL migration.
func migrate(database *types.Databases) error {
	if _, err := database.PostgresDB.Exec(database.DBCtx, fmt.Sprintf(dropTable, oldSchema.SensorUpgradeConfigsTableName)); err != nil {
		return errors.Wrapf(err, "unable to drop table %q", oldSchema.SensorUpgradeConfigsTableName)
	}
	return nil
}
