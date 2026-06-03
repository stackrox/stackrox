package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stackrox/rox/clair-adapter/datastore"
)

//go:embed migrations/indexer/01-init.sql
var indexerInitSchema string

// indexerMetadataStore implements datastore.IndexerMetadataStore for PostgreSQL.
type indexerMetadataStore struct {
	pool *pgxpool.Pool
}

// NewIndexerMetadataStore creates a new IndexerMetadataStore and initializes the schema.
func NewIndexerMetadataStore(ctx context.Context, pool *pgxpool.Pool) (datastore.IndexerMetadataStore, error) {
	store := &indexerMetadataStore{pool: pool}

	// Initialize schema
	if _, err := pool.Exec(ctx, indexerInitSchema); err != nil {
		return nil, fmt.Errorf("failed to initialize indexer metadata schema: %w", err)
	}

	return store, nil
}

// StoreManifest records or updates a manifest with its expiration time.
func (s *indexerMetadataStore) StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error {
	query := `
		INSERT INTO manifest_metadata (manifest_id, expiration)
		VALUES ($1, $2)
		ON CONFLICT (manifest_id)
		DO UPDATE SET expiration = EXCLUDED.expiration
	`

	_, err := s.pool.Exec(ctx, query, manifestID, expiration)
	if err != nil {
		return fmt.Errorf("failed to store manifest metadata: %w", err)
	}

	return nil
}

// ManifestExists checks if a manifest is present in the metadata store.
func (s *indexerMetadataStore) ManifestExists(ctx context.Context, manifestID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM manifest_metadata WHERE manifest_id = $1)`

	var exists bool
	err := s.pool.QueryRow(ctx, query, manifestID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check manifest existence: %w", err)
	}

	return exists, nil
}

// GCManifests deletes up to limit expired manifests older than the given expiration time.
// Returns the list of deleted manifest IDs.
func (s *indexerMetadataStore) GCManifests(ctx context.Context, expiration time.Time, limit int) ([]string, error) {
	query := `
		DELETE FROM manifest_metadata
		WHERE manifest_id IN (
			SELECT manifest_id
			FROM manifest_metadata
			WHERE expiration < $1
			ORDER BY expiration
			LIMIT $2
		)
		RETURNING manifest_id
	`

	rows, err := s.pool.Query(ctx, query, expiration, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to garbage collect manifests: %w", err)
	}
	defer rows.Close()

	var deletedIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan deleted manifest ID: %w", err)
		}
		deletedIDs = append(deletedIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deleted manifest IDs: %w", err)
	}

	return deletedIDs, nil
}

var _ datastore.IndexerMetadataStore = (*indexerMetadataStore)(nil)
