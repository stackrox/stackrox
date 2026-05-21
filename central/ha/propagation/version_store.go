package propagation

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var log = logging.LoggerForModule()

// VersionStore manages the global policy version counter in PostgreSQL.
type VersionStore struct {
	db postgres.DB
}

// NewVersionStore creates a new version store.
func NewVersionStore(db postgres.DB) *VersionStore {
	return &VersionStore{db: db}
}

// Initialize creates the policy_version table and inserts the initial row.
func (s *VersionStore) Initialize(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS policy_version (
			id      INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			version BIGINT NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return errors.Wrap(err, "creating policy_version table")
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO policy_version (id, version) VALUES (1, 0)
		ON CONFLICT (id) DO NOTHING
	`)
	return errors.Wrap(err, "inserting initial policy version")
}

// GetVersion returns the current global policy version.
func (s *VersionStore) GetVersion(ctx context.Context) (int64, error) {
	var version int64
	err := s.db.QueryRow(ctx, `
		SELECT version FROM policy_version WHERE id = 1
	`).Scan(&version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("policy version row not initialized")
		}
		return 0, errors.Wrap(err, "getting policy version")
	}
	return version, nil
}

// IncrementVersion atomically increments the global policy version by 1.
func (s *VersionStore) IncrementVersion(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		UPDATE policy_version SET version = version + 1 WHERE id = 1
	`)
	return errors.Wrap(err, "incrementing policy version")
}
