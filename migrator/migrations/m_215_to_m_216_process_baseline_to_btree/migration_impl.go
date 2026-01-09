package m215tom216

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	tableName   = "process_baselines"
	index       = "processbaselines_key_deploymentid"
	tmpIndex    = "processbaselines_key_deploymentid_tmp"
	indexColumn = "key_deploymentid"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// We are simply changing the index type from hash to btree if the process_baselines
	// indexes for deployment id are still hash.
	tx, err := database.PostgresDB.Begin(database.DBCtx)
	if err != nil {
		return err
	}
	ctx := postgres.ContextWithTx(database.DBCtx, tx)

	// Purposefully doing this one at a time like this to be very specific on what we are doing.
	err = indexhelper.MigrateIndex(ctx, database.PostgresDB, tableName, index, indexColumn, tmpIndex)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, errors.Wrapf(err, "unable to update process baseline indexes in migration %d", startSeqNum))
	}

	log.Info("Process baseline indexes migration complete")
	return nil
}
