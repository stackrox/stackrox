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

// ErrDidNotUpdateRow indicates the row was not updated.
var ErrDidNotUpdateRow = errors.New("determined to not update row")

// StoreIndexReport stores an external index report, indexReport, with an ID
// based on hashID with expiration. On hashID conflict, StoreIndexReport will
// use versionCmpFn to determine whether to update the row during the
// transaction. If hashID does not currently exist in the external index report
// datastore, it will be created.
func (e *externalIndexStore) StoreIndexReport(
	ctx context.Context,
	hashID string,
	indexerVersion string,
	indexReport *claircore.IndexReport,
	expiration time.Time,
	shouldUpdateStoredReportFn func(iv string) bool,
) error {
	ctx = zlog.ContextWithValues(
		ctx,
		"component",
		"datastore/postgres/externalIndexStore.StoreIndexReport",
	)

	const selectIndexReport = `
		SELECT indexer_version FROM external_index_report
		WHERE hash_id = $1 FOR UPDATE`

	var storedIndexerVersion string
	err := e.pool.QueryRow(ctx, selectIndexReport, hashID).Scan(&storedIndexerVersion)
	if err != nil {
		// If no rows were found, create the external index report.
		if errors.Is(err, pgx.ErrNoRows) {
			const insertIndexReport = `
				INSERT INTO external_index_report (hash_id, indexer_version, index_report, expiration) VALUES
					($1, $2, $3, $4)`
			_, err := e.pool.Exec(
				ctx,
				insertIndexReport,
				hashID,
				indexerVersion,
				indexReport,
				expiration.UTC(),
			)
			if err != nil {
				return fmt.Errorf("storing new external index report: %w", err)
			}
			return nil
		}

		return fmt.Errorf("querying external index reports: %w", err)
	}

	if !shouldUpdateStoredReportFn(storedIndexerVersion) {
		return fmt.Errorf("stored index report was produced with more recent indexer: %w", ErrDidNotUpdateRow)
	}

	const updateIndexReport = `
		UPDATE external_index_report SET (indexer_version, index_report, expiration, updated_at) =
			($2, $3, $4, DEFAULT) WHERE hash_id = $1`

	_, err = e.pool.Exec(ctx, updateIndexReport, hashID, indexerVersion, indexReport, expiration.UTC())
	if err != nil {
		return fmt.Errorf("updating index report: %w", err)
	}

	return nil
}

func (e *externalIndexStore) GCIndexReports(
	ctx context.Context,
	expiration time.Time,
	opts ...ReindexGCOption,
) ([]string, error) {
	o := makeReindexGCOpts(opts)

	ctx = zlog.ContextWithValues(
		ctx,
		"component",
		"datastore/postgres/externalIndexStore.GCIndexReports",
	)

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
