package m230tom231

import (
	"fmt"

	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	tableName = "auth_machine_to_machine_configs"

	// uniqueConstraint is the name GORM assigns (NamingStrategy.UniqueName) to the
	// UNIQUE constraint generated from the `unique` tag on the issuer column.
	// Multiple m2m configurations with the same issuer are now allowed, as the
	// token exchange configuration is no longer identified solely by the issuer.
	uniqueConstraint = "uni_auth_machine_to_machine_configs_issuer"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// The constraint was created inline by GORM as
	// `CONSTRAINT uni_auth_machine_to_machine_configs_issuer UNIQUE (issuer)`.
	// Dropping the constraint drops its backing index of the same name as well.
	dropConstraintStmt := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", tableName, uniqueConstraint)
	if _, err := database.PostgresDB.Exec(database.DBCtx, dropConstraintStmt); err != nil {
		return fmt.Errorf("dropping unique constraint %s on table %s: %w", uniqueConstraint, tableName, err)
	}

	// In case the uniqueness was ever enforced by a standalone index rather than
	// a constraint, drop any leftover index too. This is a no-op when the
	// constraint above existed and was dropped.
	dropIndexStmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", uniqueConstraint)
	if _, err := database.PostgresDB.Exec(database.DBCtx, dropIndexStmt); err != nil {
		return fmt.Errorf("dropping index %s on table %s: %w", uniqueConstraint, tableName, err)
	}

	log.Infof("Dropped unique constraint %s on table %s", uniqueConstraint, tableName)
	return nil
}
