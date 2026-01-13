package m214tom215

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	tableName          = "process_indicators"
	podIndex           = "processindicators_poduid"
	tmpPodIndex        = "processindicators_poduid_tmp"
	podColumn          = "poduid"
	deploymentIndex    = "processindicators_deploymentid"
	tmpDeploymentIndex = "processindicators_deploymentid_tmp"
	deploymentColumn   = "deploymentid"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// We are simply changing the index type from hash to btree if the process_indicators
	// indexes for deployment id and poduid are still hash.  There is at least one instance
	// in the field where the indexes have already been moved to btree, we do not want to
	// force that instance to remigrate.  So we will not simply drop and re-add, but
	// verify index is hash, drop old index, re-add it as btree.
	tx, err := database.PostgresDB.Begin(database.DBCtx)
	if err != nil {
		return err
	}
	ctx := postgres.ContextWithTx(database.DBCtx, tx)

	// Purposefully doing this one at a time like this to be very specific on what we are doing.
	err = indexhelper.MigrateIndex(ctx, database.PostgresDB, tableName, deploymentIndex, deploymentColumn, tmpDeploymentIndex)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = indexhelper.MigrateIndex(ctx, database.PostgresDB, tableName, podIndex, podColumn, tmpPodIndex)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, errors.Wrapf(err, "unable to update process indicator indexes in migration %d", startSeqNum))
	}

	log.Info("Process indicator indexes migration complete")
	return nil
}
