package m214tom215

import (
	"context"
	"fmt"

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

	dropIndex   = "DROP INDEX IF EXISTS %s"
	createIndex = "CREATE INDEX IF NOT EXISTS %s ON %s USING BTREE (%s)"
	renameIndex = "ALTER INDEX IF EXISTS %s RENAME TO %s"
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
	err = migrateIndex(ctx, database.PostgresDB, deploymentIndex, deploymentColumn, tmpDeploymentIndex)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = migrateIndex(ctx, database.PostgresDB, podIndex, podColumn, tmpPodIndex)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return postgreshelper.WrapRollback(ctx, tx, errors.Wrapf(err, "unable to update process indicator indexes in migraiton %d", startSeqNum))
	}

	log.Infof("Process indicator indexes migration complete")
	return nil
}

func migrateIndex(dbCtx context.Context, db postgres.DB, indexName string, indexColumn string, tmpIndexName string) error {
	log.Infof("Migrating %s index %s on column %s to btree", tableName, indexName, indexColumn)
	ctx, cancel := context.WithTimeout(dbCtx, types.DefaultMigrationTimeout)
	defer cancel()

	// Check if the hash index exists.  If it does not, we can fall through as either
	// the index already exists in the desired type in which case we have nothing to do
	// OR the index does not exist at all in which case Gorm will take care of it when
	// all schemas are applied at the end.
	exists, err := indexhelper.IndexExists(ctx, db, tableName, indexName, "hash")
	if err != nil {
		return err
	}

	if !exists {
		log.Infof("Migration of index %s is not required", indexName)
		return nil
	}

	// To minimize time without an index and to care for the event of failure forcing a
	// rollback which would recreate the hash index if failure occurred after the drop
	// we are going to create a temporary new one, drop the old one, and then rename the
	// temporary one.

	// create the new index
	createStatement := fmt.Sprintf(createIndex, tmpIndexName, tableName, indexColumn)
	_, err = db.Exec(ctx, createStatement)
	if err != nil {
		return errors.Wrapf(err, "unable to create index %s", tmpIndexName)
	}

	// drop the index
	dropStatement := fmt.Sprintf(dropIndex, indexName)
	_, err = db.Exec(ctx, dropStatement)
	if err != nil {
		return errors.Wrapf(err, "unable to drop index %s", indexName)
	}

	// rename the tmp index to be the old index
	renameStatement := fmt.Sprintf(renameIndex, tmpIndexName, indexName)
	_, err = db.Exec(ctx, renameStatement)
	if err != nil {
		return errors.Wrapf(err, "unable to rename index %s to %s", tmpIndexName, indexName)
	}

	log.Infof("Migration of index %s is complete", indexName)
	return nil
}
