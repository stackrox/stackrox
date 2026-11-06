package m227tom228

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
)

func migrate(database *types.Databases) error {
	// policies_pkey already indexes id. Drop the duplicate without blocking writes
	// from an old Central that may still be running during a rolling upgrade.
	_, err := database.PostgresDB.Exec(database.DBCtx, "DROP INDEX CONCURRENTLY IF EXISTS policies_id")
	return errors.Wrap(err, "dropping duplicate policies_id index")
}
