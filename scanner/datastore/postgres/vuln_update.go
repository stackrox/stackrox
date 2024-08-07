package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/quay/zlog"
)

const SingleBundleUpdateKey = `last-vuln-update`

// GetLastVulnerabilityUpdate implements MatcherMetadataStore.GetLastVulnerabilityUpdate.
//
// Assumes the last update is the oldest update timestamp in the table.
func (m *matcherMetadataStore) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	const selectTimestamp = `
		SELECT update_timestamp
		FROM last_vuln_update
		ORDER BY update_timestamp LIMIT 1;`
	row := m.pool.QueryRow(ctx, selectTimestamp)
	var t time.Time
	err := row.Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// SetLastVulnerabilityUpdate implements MatcherMetadataStore.SetLastVulnerabilityUpdate.
//
// We use one row for each vulnerability bundle, keyed by their name.
func (m *matcherMetadataStore) SetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) error {
	const insertTimestamp = `
		INSERT INTO last_vuln_update (key, update_timestamp)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET update_timestamp = $2;`
	_, err := m.pool.Exec(ctx, insertTimestamp, bundle, sanitizeTimestamp(lastUpdate))
	if err != nil {
		return err
	}

	return nil
}

// GetOrSetLastVulnerabilityUpdate implements MatcherMetadataStore.GetOrSetLastVulnerabilityUpdate
func (m *matcherMetadataStore) GetOrSetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) (time.Time, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Try inserting a new row with the provided timestamp, in case the row does not exist.
	const insertIfNotExist = `
		INSERT INTO last_vuln_update (key, update_timestamp)
		VALUES ($1, $2)
		ON CONFLICT (key) DO NOTHING;`
	_, err = m.pool.Exec(ctx, insertIfNotExist, bundle, sanitizeTimestamp(lastUpdate))
	if err != nil {
		return time.Time{}, err
	}

	// Get the timestamp, whether it's just been inserted or already existed.
	const getTimestamp = `SELECT update_timestamp FROM last_vuln_update WHERE key = $1;`
	var t time.Time
	err = m.pool.QueryRow(ctx, getTimestamp, bundle).Scan(&t)
	if err != nil {
		return time.Time{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return time.Time{}, err
	}

	return t.UTC(), nil
}

// GCVulnerabilityUpdates implements MatcherMetadataStore.GCVulnerabilityUpdates
//
// Removes all entries for inactive vulnerability bundles that are older than the
// last know update.
func (m *matcherMetadataStore) GCVulnerabilityUpdates(ctx context.Context, activeUpdaters []string, lastUpdate time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/matcherMetadataStore.GCVulnerabilityUpdates")
	const deleteUnknownAndInactive = `
		DELETE FROM last_vuln_update
		WHERE NOT key = ANY($1) AND update_timestamp < $2
		RETURNING key`
	rows, err := m.pool.Query(ctx, deleteUnknownAndInactive, activeUpdaters, sanitizeTimestamp(lastUpdate))
	if err != nil {
		return err
	}
	defer rows.Close()
	var deletedRows []string
	for rows.Next() {
		var deletedRow string
		err := rows.Scan(&deletedRow)
		if err != nil {
			zlog.Warn(ctx).Err(err).Msg("scanning deleted row")
			continue
		}
		deletedRows = append(deletedRows, deletedRow)
	}
	if err := rows.Err(); err != nil {
		zlog.Warn(ctx).Err(err).Msg("reading deleted rows")
	}
	if len(deletedRows) > 0 {
		zlog.Info(ctx).Strs("deleted_bundles", deletedRows).Msg("deleted inactive vulnerability bundle(s)")
	}
	return nil
}

func sanitizeTimestamp(lastUpdate time.Time) time.Time {
	return lastUpdate.UTC().Truncate(time.Second)
}
