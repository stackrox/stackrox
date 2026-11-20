package m228tom229

import (
	"fmt"

	"github.com/pkg/errors"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_228_to_m_229_drop_compliance_v1_tables/schema/old"
	"github.com/stackrox/rox/migrator/types"
)

var complianceV1Tables = []string{
	oldSchema.ComplianceConfigsTableName,
	oldSchema.ComplianceDomainsTableName,
	oldSchema.ComplianceStringsTableName,
	oldSchema.ComplianceRunResultsTableName,
	oldSchema.ComplianceRunMetadataTableName,
}

// migrate drops the five orphaned Compliance 1.0 Postgres tables. The Compliance 1.0
// engine has been removed, so nothing reads or writes these tables anymore. GORM
// AutoMigrate never drops tables, hence this explicit DDL migration.
func migrate(database *types.Databases) error {
	for _, table := range complianceV1Tables {
		if _, err := database.PostgresDB.Exec(database.DBCtx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return errors.Wrapf(err, "unable to drop table %q", table)
		}
	}
	return nil
}
