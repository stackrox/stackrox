package m222tom223

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	tableName   = "deployments"
	indexName   = "deployments_tombstone_expiresat"
	indexColumn = "tombstone_expiresat"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	// Add a btree index on tombstone_expiresat to make pruning expired deployments efficient.
	// The pruner will query for deployments WHERE tombstone_expiresat < NOW().
	tx, err := database.PostgresDB.Begin(database.DBCtx)
	if err != nil {
		return err
	}
	ctx := postgres.ContextWithTx(database.DBCtx, tx)

	// Create the index concurrently to avoid blocking other operations.
	// Note: CONCURRENTLY cannot be used inside a transaction, so we'll create it normally.
	// The migration framework already locks appropriately.
	_, err = tx.Exec(ctx, "CREATE INDEX IF NOT EXISTS "+indexName+" ON "+tableName+" USING BTREE ("+indexColumn+")")
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, errors.Wrapf(err, "failed to create index %s on %s(%s)", indexName, tableName, indexColumn))
	}

	if err = tx.Commit(ctx); err != nil {
		return postgreshelper.WrapRollback(ctx, tx, errors.Wrapf(err, "failed to commit transaction for migration %d", startSeqNum))
	}

	log.Infof("Successfully created index %s on %s.%s", indexName, tableName, indexColumn)
	return nil
}
