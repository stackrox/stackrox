package leases

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/postgres"
)

// Lease represents a sensor connection lease held by a Central pod.
type Lease struct {
	ClusterID     string
	PodID         string
	ConnectedAt   time.Time
	LastHeartbeat time.Time
}

// Store manages sensor connection leases in PostgreSQL.
type Store struct {
	db postgres.DB
}

// New creates a new lease store.
func New(db postgres.DB) *Store {
	return &Store{db: db}
}

// Initialize creates the sensor_connections table if it doesn't exist.
func (s *Store) Initialize(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sensor_connections (
			cluster_id     TEXT PRIMARY KEY,
			pod_id         TEXT NOT NULL,
			connected_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return errors.Wrap(err, "creating sensor_connections table")
}

// Claim registers or updates a sensor connection lease.
func (s *Store) Claim(ctx context.Context, clusterID, podID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO sensor_connections (cluster_id, pod_id, connected_at, last_heartbeat)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (cluster_id) DO UPDATE
		SET pod_id = $2, connected_at = NOW(), last_heartbeat = NOW()
	`, clusterID, podID)
	return errors.Wrap(err, "claiming sensor connection")
}

// Release removes a sensor connection lease.
func (s *Store) Release(ctx context.Context, clusterID, podID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM sensor_connections
		WHERE cluster_id = $1 AND pod_id = $2
	`, clusterID, podID)
	return errors.Wrap(err, "releasing sensor connection")
}

// Heartbeat updates the last_heartbeat timestamp.
func (s *Store) Heartbeat(ctx context.Context, clusterID, podID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE sensor_connections
		SET last_heartbeat = NOW()
		WHERE cluster_id = $1 AND pod_id = $2
	`, clusterID, podID)
	return errors.Wrap(err, "heartbeat sensor connection")
}

// GetHolder returns the pod ID holding the lease for a cluster.
func (s *Store) GetHolder(ctx context.Context, clusterID string) (string, error) {
	var podID string
	err := s.db.QueryRow(ctx, `
		SELECT pod_id FROM sensor_connections WHERE cluster_id = $1
	`, clusterID).Scan(&podID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", errors.Wrap(err, "getting connection holder")
	}
	return podID, nil
}

// GetStaleLeases returns leases with heartbeats older than threshold.
func (s *Store) GetStaleLeases(ctx context.Context, threshold time.Duration) ([]Lease, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cluster_id, pod_id, connected_at, last_heartbeat
		FROM sensor_connections
		WHERE last_heartbeat < NOW() - $1 * interval '1 second'
	`, threshold.Seconds())
	if err != nil {
		return nil, errors.Wrap(err, "getting stale leases")
	}
	defer rows.Close()

	var leases []Lease
	for rows.Next() {
		var l Lease
		if err := rows.Scan(&l.ClusterID, &l.PodID, &l.ConnectedAt, &l.LastHeartbeat); err != nil {
			return nil, errors.Wrap(err, "scanning lease")
		}
		leases = append(leases, l)
	}
	return leases, rows.Err()
}

// GetAllLeases returns all active leases.
func (s *Store) GetAllLeases(ctx context.Context) ([]Lease, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cluster_id, pod_id, connected_at, last_heartbeat
		FROM sensor_connections
		ORDER BY connected_at
	`)
	if err != nil {
		return nil, errors.Wrap(err, "getting all leases")
	}
	defer rows.Close()

	var leases []Lease
	for rows.Next() {
		var l Lease
		if err := rows.Scan(&l.ClusterID, &l.PodID, &l.ConnectedAt, &l.LastHeartbeat); err != nil {
			return nil, errors.Wrap(err, "scanning lease")
		}
		leases = append(leases, l)
	}
	return leases, rows.Err()
}
