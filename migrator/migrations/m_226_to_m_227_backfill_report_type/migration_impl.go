package m226tom227

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	migrationStmt = `
		ALTER TABLE report_snapshots ADD COLUMN IF NOT EXISTS type INTEGER DEFAULT 0;
		UPDATE report_snapshots SET type = 0 WHERE type IS NULL;
	`
)

func migrate(database *types.Databases) error {
	ctx := context.Background()

	log.Info("Adding type column and backfilling report_snapshots.type: setting type=0 for existing reports")

	// Execute both ALTER TABLE and UPDATE in a single statement
	// This is idempotent - safe to re-run:
	// - ADD COLUMN IF NOT EXISTS won't fail if column exists
	// - UPDATE WHERE type IS NULL won't modify rows already set to 0
	_, err := database.PostgresDB.Exec(ctx, migrationStmt)
	if err != nil {
		return errors.Wrap(err, "failed to add type column and backfill report_snapshots")
	}

	log.Info("Successfully added type column and backfilled existing report snapshots with type=0")

	return nil
}
