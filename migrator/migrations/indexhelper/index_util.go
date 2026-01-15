package indexhelper

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	indexQuery = `SELECT EXISTS(
	SELECT tab.relname, idx.relname, am.amname
	FROM pg_index x
	JOIN pg_class idx ON idx.oid=x.indexrelid
	JOIN pg_class tab ON tab.oid=x.indrelid
	JOIN pg_am am ON am.oid=idx.relam
	WHERE tab.relname = $1 AND
	idx.relname = $2 AND am.amname = $3
	)`

	dropIndex   = "DROP INDEX IF EXISTS %s"
	createIndex = "CREATE INDEX IF NOT EXISTS %s ON %s USING BTREE (%s)"
	renameIndex = "ALTER INDEX IF EXISTS %s RENAME TO %s"
)

var (
	log = logging.LoggerForModule()
)

// IndexExists returns if an index on a given table with a given name and type exists.
// This could have been more generic, but in the migrator it is best to be very explicit
// on what we are working with.
func IndexExists(ctx context.Context, db postgres.DB, tableName, indexName, indexType string) (bool, error) {
	row := db.QueryRow(ctx, indexQuery, tableName, indexName, indexType)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

// MigrateIndex is a helper that specifically converts a hash index to btree.  Limiting in scope to that
// for now.
func MigrateIndex(dbCtx context.Context, db postgres.DB, tableName string, indexName string, indexColumn string, tmpIndexName string) error {
	log.Infof("Migrating %s index %s on column %s to btree", tableName, indexName, indexColumn)
	ctx, cancel := context.WithTimeout(dbCtx, types.DefaultMigrationTimeout)
	defer cancel()

	// Check if the hash index exists.  If it does not, we can fall through as either
	// the index already exists in the desired type in which case we have nothing to do
	// OR the index does not exist at all in which case Gorm will take care of it when
	// all schemas are applied at the end.
	exists, err := IndexExists(ctx, db, tableName, indexName, "hash")
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
