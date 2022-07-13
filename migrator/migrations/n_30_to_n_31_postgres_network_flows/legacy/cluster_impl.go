package legacy

import (
	"context"

	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/common"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/store"
	"github.com/stackrox/rox/pkg/rocksdb"
)

// NewClusterStore returns a new ClusterStore instance using the provided rocksdb instance.
func NewClusterStore(db *rocksdb.RocksDB) store.ClusterStore {
	return &clusterStoreImpl{
		db: db,
	}
}

type clusterStoreImpl struct {
	db *rocksdb.RocksDB
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	return &flowStoreImpl{
		db:        s.db,
		keyPrefix: common.FlowStoreKeyPrefix(clusterID),
	}
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(ctx context.Context, clusterID string) (store.FlowStore, error) {
	fs := &flowStoreImpl{
		db:        s.db,
		keyPrefix: common.FlowStoreKeyPrefix(clusterID),
	}
	return fs, nil
}
