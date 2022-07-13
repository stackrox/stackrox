package postgres

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/store"
)

// NewClusterStore returns a new ClusterStore instance using the provided rocksdb instance.
func NewClusterStore(db *pgxpool.Pool) store.ClusterStore {
	return &clusterStoreImpl{
		db: db,
	}
}

type clusterStoreImpl struct {
	db *pgxpool.Pool
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	return &flowStoreImpl{
		db:        s.db,
		clusterID: clusterID,
	}
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(_ context.Context, clusterID string) (store.FlowStore, error) {
	return New(s.db, clusterID), nil
}
