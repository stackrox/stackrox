package m216tom217

import (
	"fmt"

	"github.com/pkg/errors"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_216_to_m_217_remove_compliance_benchmark_table/schema/old"
	"github.com/stackrox/rox/migrator/types"
)

const dropTable = "DROP TABLE IF EXISTS %s"

func migrate(database *types.Databases) error {
	_, err := database.PostgresDB.Exec(database.DBCtx, fmt.Sprintf(dropTable, oldSchema.ComplianceOperatorBenchmarkV2ProfilesTableName))
	if err != nil {
		return errors.Wrapf(err, "unable to drop table %q", oldSchema.ComplianceOperatorBenchmarkV2ProfilesTableName)
	}

	_, err = database.PostgresDB.Exec(database.DBCtx, fmt.Sprintf(dropTable, oldSchema.ComplianceOperatorBenchmarkV2TableName))
	if err != nil {
		return errors.Wrapf(err, "unable to drop table %q", oldSchema.ComplianceOperatorBenchmarkV2TableName)
	}

	return nil
}
