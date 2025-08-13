package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
)

func (e *externalIndexStore) StoreIndexReport(ctx context.Context, hashID string, clusterName string, indexReport *claircore.IndexReport) error {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/externalIndexStore.StoreIndexReport")

	const insertIndexReport = `
		INSERT INTO external_index_report (hash_id, clusterName, indexReport) VALUES
			($1, $2, $3)
		ON CONFLICT (hash_id, clusterName) DO UPDATE SET indexReport = $3`

	_, err := e.pool.Exec(ctx, hashID, clusterName, indexReport)
	if err != nil {
		return err
	}

	return nil
}

func (e *externalIndexStore) StoreIndexReportWithExpiration(ctx context.Context, hashID string, clusterName string, indexReport *claircore.IndexReport, expiration time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/externalIndexStore.StoreIndexReportWithExpiration")

	const insertIndexReport = `
		INSERT INTO external_index_report (hash_id, clusterName, indexReport, expiration) VALUES
			($1, $2, $3, $4)
		ON CONFLICT (hash_id, clusterName) DO UPDATE SET
			indexReport = $3,
			expiration = $4`

	_, err := e.pool.Exec(ctx, hashID, clusterName, indexReport, expiration)
	if err != nil {
		return err
	}

	return nil
}

func (e *externalIndexStore) GCIndexReports(ctx context.Context, expiration time.Time, opts ...ReindexGCOption) ([]string, error) {
	o := makeReindexGCOpts(opts)

	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/externalIndexStore.GCIndexReports")

	const deleteIndexReports = `
		DELETE FROM external_index_report
		WHERE hash_id IN (
		    SELECT hash_id FROM external_index_report WHERE expiration < $1 LIMIT $2
		)
		RETURNING hash_id`

	// Make this a transaction, as failure to delete the index report should stop deletion.
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning ReindexGC transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Delete expired rows from external_index_report
	rows, err := tx.Query(ctx, deleteIndexReports, expiration.UTC(), o.gcThrottle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deletedHashes []string
	for rows.Next() {
		var hashID string
		if err := rows.Scan(&hashID); err != nil {
			zlog.Warn(ctx).Err(err).Msg("scanning deleted external index report row")
			continue
		}
		deletedHashes = append(deletedHashes, hashID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading deleted external index report rows: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing deleted external index reports: %w", err)
	}

	return deletedHashes, nil
}
