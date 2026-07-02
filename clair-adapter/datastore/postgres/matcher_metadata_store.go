package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stackrox/rox/clair-adapter/datastore"
)

//go:embed migrations/matcher/01-init.sql
var matcherInitSchema string

//go:embed migrations/matcher/02-update-timestamp.sql
var matcherTimestampMigration string

// matcherMetadataStore implements datastore.MatcherMetadataStore for PostgreSQL.
type matcherMetadataStore struct {
	pool *pgxpool.Pool
}

// NewMatcherMetadataStore creates a new MatcherMetadataStore and initializes the schema.
func NewMatcherMetadataStore(ctx context.Context, pool *pgxpool.Pool) (datastore.MatcherMetadataStore, error) {
	store := &matcherMetadataStore{pool: pool}

	// Initialize schema - run both migrations in order
	if _, err := pool.Exec(ctx, matcherInitSchema); err != nil {
		return nil, fmt.Errorf("failed to initialize matcher metadata schema: %w", err)
	}

	if _, err := pool.Exec(ctx, matcherTimestampMigration); err != nil {
		return nil, fmt.Errorf("failed to apply matcher timestamp migration: %w", err)
	}

	return store, nil
}

// GetLastVulnerabilityUpdate returns the earliest vulnerability update timestamp across all bundles.
// Returns zero time if no updates have been recorded.
func (s *matcherMetadataStore) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	query := `SELECT COALESCE(MIN(update_timestamp), to_timestamp(0)) FROM last_vuln_update`

	var lastUpdate time.Time
	err := s.pool.QueryRow(ctx, query).Scan(&lastUpdate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last vulnerability update: %w", err)
	}

	return lastUpdate, nil
}

// SetLastVulnerabilityUpdate records the update timestamp for a specific vulnerability bundle.
func (s *matcherMetadataStore) SetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) error {
	query := `
		INSERT INTO last_vuln_update (key, timestamp, update_timestamp)
		VALUES ($1, $2, $3)
		ON CONFLICT (key)
		DO UPDATE SET
			timestamp = EXCLUDED.timestamp,
			update_timestamp = EXCLUDED.update_timestamp
	`

	// Store both text timestamp for backwards compatibility and proper timestamp column
	timestampText := lastUpdate.Format(time.RFC3339)

	_, err := s.pool.Exec(ctx, query, bundle, timestampText, lastUpdate)
	if err != nil {
		return fmt.Errorf("failed to set last vulnerability update: %w", err)
	}

	return nil
}

var _ datastore.MatcherMetadataStore = (*matcherMetadataStore)(nil)
