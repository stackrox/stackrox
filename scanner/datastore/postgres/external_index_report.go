package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
)

func (e *externalIndexStore) StoreIndexReport(ctx context.Context, hashID string, indexReport *claircore.IndexReport, expiration time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/externalIndexStore.StoreIndexReport")

	const insertIndexReport = `
		INSERT INTO external_index_report (hash_id, indexReport, expiration) VALUES
			($1, $2, $3)
		ON CONFLICT (hash_id) DO UPDATE SET
			indexReport = $2,
			expiration = $3`

	_, err := e.pool.Exec(ctx, hashID, indexReport, expiration)
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

	// There may be multiple instances of scanner attempting to GC these index
	// reports, so use a transaction to ensure that the delete operation is
	// atomic. If there's an error during the transaction, we'll still commit
	// what's been marked for deletion so far since there isn't a reason to
	// rollback. Note that if there was an error during the transaction, we
	// only return that error and only log any errors from committing the
	// transaction.
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning GCIndexReports transaction: %w", err)
	}
	defer func() {
		if commitErr := tx.Commit(ctx); commitErr != nil && !errors.Is(commitErr, pgx.ErrTxClosed) {
			zlog.Warn(ctx).Err(commitErr).Msg("failed to commit GCIndexReports transaction")
		}
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
