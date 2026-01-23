package m220tom221

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = loghelper.LogWrapper{}

	// All tables that have the Traits message with expires_at field.
	tablesWithTraits = []string{
		"roles",
		"permission_sets",
		"simple_access_scopes",
		"auth_providers",
		"auth_machine_to_machine_configs",
		"signature_integrations",
		"notifiers",
		"groups",
	}
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	for _, table := range tablesWithTraits {
		if err := addTraitsExpiresAtColumn(ctx, database.PostgresDB, table); err != nil {
			return errors.Wrapf(err, "failed to add traits_expires_at column to table %s", table)
		}
	}

	return nil
}

func addTraitsExpiresAtColumn(ctx context.Context, db postgres.DB, table string) error {
	ctx, cancel := context.WithTimeout(ctx, types.DefaultMigrationTimeout)
	defer cancel()

	alterTableStmt := fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN IF NOT EXISTS traits_expires_at TIMESTAMP WITHOUT TIME ZONE;",
		table,
	)

	_, err := db.Exec(ctx, alterTableStmt)
	if err != nil {
		return errors.Wrapf(err, "unable to alter table %s", table)
	}

	log.WriteToStderrf("Added traits_expires_at column to table %s", table)
	return nil
}
