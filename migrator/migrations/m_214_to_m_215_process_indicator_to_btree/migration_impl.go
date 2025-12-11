package m214tom215

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	hashIndexQuery = `SELECT EXISTS(
	SELECT tab.relname, cls.relname, am.amname
	FROM pg_index idx
	JOIN pg_class cls ON cls.oid=idx.indexrelid
	JOIN pg_class tab ON tab.oid=idx.indrelid
	JOIN pg_am am ON am.oid=cls.relam
	where tab.relname = $1 AND
	am.amname = 'hash' AND cls.relname = $2
	)`

	tableName        = "process_indicators"
	podIndex         = "processindicators_poduid"
	podColumn        = "poduid"
	deploymentIndex  = "processindicators_deploymentid"
	deploymentColumn = "deployment_id"

	dropIndex   = "DROP INDEX if exists %s"
	createIndex = "CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s USING BTREE (%s)"
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
	log.Infof("Process indicator index migration complete")

	// Purposefully doing this one at a time like this to be very specific on what we are doing.
	err := migrateIndex(database.DBCtx, database.PostgresDB, deploymentIndex, deploymentColumn)
	if err != nil {
		return err
	}

	err = migrateIndex(database.DBCtx, database.PostgresDB, podIndex, podColumn)
	if err != nil {
		return err
	}

	log.Infof("Process indicator index migration complete")
	return nil
}

func migrateIndex(dbCtx context.Context, db postgres.DB, indexName string, indexColumn string) error {
	log.Infof("Migrating %s index %s on column %s to btree", tableName, indexName, indexColumn)
	ctx, cancel := context.WithTimeout(dbCtx, types.DefaultMigrationTimeout)
	defer cancel()

	row := db.QueryRow(ctx, hashIndexQuery, tableName, indexName)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return err
	}

	if !exists {
		return nil
	}

	// drop the index
	dropStatement := fmt.Sprintf(dropIndex, indexName)
	log.Infof(dropStatement)
	_, err := db.Exec(ctx, dropStatement)
	if err != nil {
		log.Error(errors.Wrapf(err, "unable to drop index %s", indexName))
	}

	// create the new index concurrently
	createStatement := fmt.Sprintf(createIndex, indexName, tableName, indexColumn)
	log.Infof(createStatement)
	_, err = db.Exec(ctx, createStatement)
	if err != nil {
		log.Error(errors.Wrapf(err, "unable to create index %s", indexName))
	}

	log.Infof("Migration of index %s is complete", indexName)
	return nil
}
