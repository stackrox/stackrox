package rocksdb

import (
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store/common"
	"github.com/tecbot/gorocksdb"
)

// NewClusterStore returns a new ClusterStore instance using the provided badger DB instance.
func NewClusterStore(db *gorocksdb.DB) store.ClusterStore {
	return &clusterStoreImpl{
		db: db,
	}
}

type clusterStoreImpl struct {
	db *gorocksdb.DB
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	return &flowStoreImpl{
		db:        s.db,
		keyPrefix: common.FlowStoreKeyPrefix(clusterID),
	}
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(clusterID string) (store.FlowStore, error) {
	fs := &flowStoreImpl{
		db:        s.db,
		keyPrefix: common.FlowStoreKeyPrefix(clusterID),
	}
	return fs, nil
}
