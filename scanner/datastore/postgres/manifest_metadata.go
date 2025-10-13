package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
)

func (i *indexerMetadataStore) MigrateManifests(ctx context.Context, expiration time.Time) ([]string, error) {
	// Though i.store is not used here, directly, it is used as a signal
	// that the manifest table is expected to exist.
	if i.store == nil {
		return nil, errors.New("indexer store not defined")
	}

	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.MigrateManifests")

	// insertMissingManifests inserts missing manifests from the manifest table into manifest_metadata,
	// and it sets the expiration time to the given expiration.
	const insertMissingManifests = `
		INSERT INTO manifest_metadata (manifest_id, expiration)
		SELECT m.hash, $1
		FROM manifest m
		WHERE NOT EXISTS (
			SELECT FROM manifest_metadata mm WHERE mm.manifest_id = m.hash
		)
		RETURNING manifest_id`

	rows, err := i.pool.Query(ctx, insertMissingManifests, expiration.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var missingManifests []string
	for rows.Next() {
		var manifestID string
		if err := rows.Scan(&manifestID); err != nil {
			zlog.Warn(ctx).Err(err).Msg("scanning manifest row")
			continue
		}
		missingManifests = append(missingManifests, manifestID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading manifest metadata rows: %w", err)
	}

	return missingManifests, nil
}

func (i *indexerMetadataStore) StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.StoreManifest")

	// insertManifest inserts the metadata into manifest_metadata, overwriting the previous expiration, if it exists.
	const insertManifest = `
		INSERT INTO manifest_metadata (manifest_id, expiration) VALUES
			($1, $2)
		ON CONFLICT (manifest_id) DO UPDATE SET expiration = $2`

	_, err := i.pool.Exec(ctx, insertManifest, manifestID, expiration.UTC())
	if err != nil {
		return err
	}

	return nil
}

func (i *indexerMetadataStore) ManifestExists(ctx context.Context, manifestID string) (bool, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.ManifestExists")

	// selectManifest returns 1 if the given manifest exists in the table.
	// If it does not exist, then no rows will be returned.
	const selectManifest = `SELECT 1 FROM manifest_metadata WHERE manifest_id = $1`

	row := i.pool.QueryRow(ctx, selectManifest, manifestID)
	var value int
	err := row.Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking if manifest exists: %w", err)
	}

	// Sanity check the returned value is what is expected.
	return value == 1, nil
}

func (i *indexerMetadataStore) GCManifests(ctx context.Context, expiration time.Time, opts ...ReindexGCOption) ([]string, error) {
	o := makeReindexGCOpts(opts)

	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.GCManifests")

	const deleteManifests = `
		DELETE FROM manifest_metadata
		WHERE manifest_id IN (
		    SELECT manifest_id FROM manifest_metadata WHERE expiration < $1 LIMIT $2
		)
		RETURNING manifest_id`

	// Make this a transaction, as failure to delete the manifest should stop the deletion of its metadata.
	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning GCManifests transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Delete expired rows from manifest_metadata
	rows, err := tx.Query(ctx, deleteManifests, expiration.UTC(), o.gcThrottle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deletedManifests []string
	for rows.Next() {
		var manifestID string
		if err := rows.Scan(&manifestID); err != nil {
			zlog.Warn(ctx).Err(err).Msg("scanning deleted manifest metadata row")
			continue
		}
		deletedManifests = append(deletedManifests, manifestID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading deleted manifest metadata rows: %w", err)
	}

	if i.store != nil {
		digs := make([]claircore.Digest, 0, len(deletedManifests))
		for _, m := range deletedManifests {
			d, err := claircore.ParseDigest(m)
			if err != nil {
				return nil, fmt.Errorf("parsing deleted manifest metadata ID: %w", err)
			}
			digs = append(digs, d)
		}
		deletedDigs, err := i.store.DeleteManifests(ctx, digs...)
		if err != nil {
			return nil, fmt.Errorf("deleting manifests: %w", err)
		}
		if len(deletedDigs) > 0 {
			digs := make([]string, 0, len(deletedDigs))
			for _, d := range deletedDigs {
				digs = append(digs, d.String())
			}
			zlog.Debug(ctx).Strs("deleted_manifests", digs).Msg("deleted manifests")
		}
		zlog.Info(ctx).Int("deleted_manifests", len(deletedDigs)).Msg("deleted manifests")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing deleted manifests: %w", err)
	}

	return deletedManifests, nil
}
