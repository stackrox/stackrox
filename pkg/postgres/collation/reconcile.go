package collation

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

// IndexInfo describes a BTREE index affected by a collation version change.
type IndexInfo struct {
	Name  string
	Table string
}

// CheckMismatch queries the database for a collation version mismatch between
// the recorded version (set at initdb time) and the version provided by the OS.
func CheckMismatch(ctx context.Context, db postgres.DB) (recorded, actual string, mismatch bool, err error) {
	const query = `
		SELECT d.datcollversion,
		       pg_catalog.pg_database_collation_actual_version(d.oid)
		FROM pg_catalog.pg_database d
		WHERE d.datname = current_database()`

	var recordedVal, actualVal *string
	if err := db.QueryRow(ctx, query).Scan(&recordedVal, &actualVal); err != nil {
		return "", "", false, fmt.Errorf("querying collation version: %w", err)
	}

	// NULL datcollversion occurs with C locale databases that have no collation version.
	if recordedVal == nil || actualVal == nil || *recordedVal == "" || *actualVal == "" {
		return "", "", false, nil
	}

	return *recordedVal, *actualVal, *recordedVal != *actualVal, nil
}

// AffectedIndexes returns BTREE indexes in the public schema that use
// locale-sensitive collation (indcollation OID != 0) and are therefore
// affected by glibc/ICU version changes.
func AffectedIndexes(ctx context.Context, db postgres.DB) ([]IndexInfo, error) {
	const query = `
		SELECT c.relname AS index_name, t.relname AS table_name
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_am am ON am.oid = c.relam
		WHERE am.amname = 'btree'
		  AND c.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		  AND EXISTS (
		      SELECT 1 FROM unnest(i.indcollation) AS colloid WHERE colloid != 0
		  )
		ORDER BY t.relname, c.relname`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying affected indexes: %w", err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		if err := rows.Scan(&idx.Name, &idx.Table); err != nil {
			return nil, fmt.Errorf("scanning index row: %w", err)
		}
		indexes = append(indexes, idx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading index rows: %w", err)
	}
	return indexes, nil
}

// Reconcile detects a collation version mismatch, reindexes all affected BTREE
// indexes, and refreshes the recorded collation version. If no mismatch exists,
// it returns nil immediately (one cheap query per startup).
func Reconcile(ctx context.Context, db postgres.DB, perIndexTimeout time.Duration) error {
	recorded, actual, mismatch, err := CheckMismatch(ctx, db)
	if err != nil {
		return fmt.Errorf("checking collation mismatch: %w", err)
	}
	if !mismatch {
		return nil
	}

	log.Infof("Collation version mismatch detected (recorded=%s, actual=%s). Reindexing affected BTREE indexes.", recorded, actual)

	indexes, err := AffectedIndexes(ctx, db)
	if err != nil {
		return fmt.Errorf("finding affected indexes: %w", err)
	}

	if len(indexes) == 0 {
		log.Warnf("Collation mismatch detected but no affected indexes found")
	} else {
		log.Infof("Found %d collation-dependent indexes to reindex", len(indexes))
		for i, idx := range indexes {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Infof("Reindexing %d/%d: %s (table: %s)", i+1, len(indexes), idx.Name, idx.Table)
			stmtCtx, cancel := context.WithTimeout(ctx, perIndexTimeout)
			_, execErr := db.Exec(stmtCtx, "REINDEX INDEX CONCURRENTLY "+pq.QuoteIdentifier(idx.Name))
			cancel()
			if execErr != nil {
				return fmt.Errorf("reindexing %s: %w", idx.Name, execErr)
			}
		}
	}

	var dbName string
	if err := db.QueryRow(ctx, "SELECT current_database()").Scan(&dbName); err != nil {
		return fmt.Errorf("querying database name: %w", err)
	}
	if _, err := db.Exec(ctx, "ALTER DATABASE "+pq.QuoteIdentifier(dbName)+" REFRESH COLLATION VERSION"); err != nil {
		return fmt.Errorf("refreshing collation version: %w", err)
	}

	log.Infof("Collation reconciliation complete. Reindexed %d indexes, collation version updated from %s to %s.", len(indexes), recorded, actual)
	return nil
}
