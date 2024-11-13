package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
)

func (i *indexerMetadataStore) Init(ctx context.Context) ([]string, error) {
	if i.store == nil {
		return nil, errors.New("indexer store not defined")
	}

	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.Init")

	const insertMissingManifests = `
		INSERT INTO manifest_metadata (manifest_id, expiration)
		SELECT m.hash, now() + (make_interval(days => 23) * random()) + make_interval(days => 7)
		FROM manifest m
		WHERE NOT EXISTS (
			SELECT FROM manifest_metadata mm WHERE mm.manifest_id = m.hash
		)
		RETURNING manifest_id`

	rows, err := i.pool.Query(ctx, insertMissingManifests)
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
		zlog.Warn(ctx).Err(err).Msg("reading manifest rows")
	}
	if len(missingManifests) > 0 {
		zlog.Debug(ctx).Strs("migrated_manifests", missingManifests).Msg("migrated missing manifest metadata")
	}
	zlog.Info(ctx).Int("migrated_manifests", len(missingManifests)).Msg("migrated missing manifest metadata")

	return missingManifests, nil
}

func (i *indexerMetadataStore) StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.StoreManifest")

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

func (i *indexerMetadataStore) GCManifests(ctx context.Context, t time.Time) ([]string, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "datastore/postgres/indexerMetadataStore.GCManifests")

	const deleteManifests = `
		DELETE FROM manifest_metadata
		WHERE expiration < $1
		RETURNING manifest_id`

	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning GCManifests transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Delete expired rows from manifest_metadata
	rows, err := tx.Query(ctx, deleteManifests, t.UTC())
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
