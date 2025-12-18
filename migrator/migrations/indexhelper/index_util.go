package indexhelper

import (
	"context"

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
