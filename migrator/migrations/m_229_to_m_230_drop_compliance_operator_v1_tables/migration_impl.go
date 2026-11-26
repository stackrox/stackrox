package m229tom230

import (
	"fmt"

	"github.com/pkg/errors"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_229_to_m_230_drop_compliance_operator_v1_tables/schema/old"
	"github.com/stackrox/rox/migrator/types"
)

var complianceOperatorV1Tables = []string{
	oldSchema.ComplianceOperatorCheckResultsTableName,
	oldSchema.ComplianceOperatorProfilesTableName,
	oldSchema.ComplianceOperatorRulesTableName,
	oldSchema.ComplianceOperatorScanSettingBindingsTableName,
	oldSchema.ComplianceOperatorScansTableName,
}

// migrate drops the five orphaned Compliance Operator v1 Postgres tables. The v1 wire
// protocol was removed (ROX-36856 commit 754c03f3bd); v2 equivalents replaced them.
// GORM AutoMigrate never drops tables, hence this explicit DDL migration.
func migrate(database *types.Databases) error {
	for _, table := range complianceOperatorV1Tables {
		if _, err := database.PostgresDB.Exec(database.DBCtx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return errors.Wrapf(err, "unable to drop table %q", table)
		}
	}
	return nil
}
