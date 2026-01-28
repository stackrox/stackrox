package m219tom220

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	tableName            = "network_flows_v2"
	indexSrcEntity       = "network_flows_src_v2"
	tmpIndexSrcEntity    = "network_flows_src_v2_tmp"
	indexSrcEntityColumn = "props_srcentity_Id"

	indexDstEntity       = "network_flows_dst_v2"
	tmpIndexDstEntity    = "network_flows_dst_v2_tmp"
	indexDstEntityColumn = "props_dstentity_Id"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// Network_flows_v2 is a partitioned table.
	// Creating the indexes on each partition and waiting on them to
	// be done before adding the index to the owning table was considered.  But
	// that would be a complicated migration and adds little value due to how our migrator
	// locks everything anyway.  The same amount of work would have to be done before the migrator
	// finished so the value of doing such work was minimal.  Additionally testing on a large set of
	// 180M network flows in a small database yielded this work to be completed in a matter of
	// minutes.  If our migrator worked on a functioning system the story would be different.  But
	// since it blocks there is no value added in the more complicated flow.

	// We are simply changing the index type from hash to btree if the network flows v2
	// indexes are still hash.
	tx, err := database.PostgresDB.Begin(database.DBCtx)
	if err != nil {
		return err
	}
	ctx := postgres.ContextWithTx(database.DBCtx, tx)

	// Purposefully doing this one at a time like this to be very specific on what we are doing.
	err = indexhelper.MigrateIndex(ctx, database.PostgresDB, tableName, indexSrcEntity, indexSrcEntityColumn, tmpIndexSrcEntity)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = indexhelper.MigrateIndex(ctx, database.PostgresDB, tableName, indexDstEntity, indexDstEntityColumn, tmpIndexDstEntity)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, errors.Wrapf(err, "unable to update network flows indexes in migration %d", startSeqNum))
	}

	log.Infof("Network flow indexes migration complete")
	return nil
}
