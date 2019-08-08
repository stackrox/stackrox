package badger

import (
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/badgerhelper"
)

const (
	// GlobalPrefix is the generic prefix for network flows
	GlobalPrefix = "networkFlows2"
)

// NewClusterStore returns a new ClusterStore instance using the provided badger DB instance.
func NewClusterStore(db *badger.DB) store.ClusterStore {
	return &clusterStoreImpl{
		db: db,
	}
}

type clusterStoreImpl struct {
	db *badger.DB
}

func flowStoreKeyPrefix(clusterID string) []byte {
	return []byte(fmt.Sprintf("%s\x00%s\x00", GlobalPrefix, clusterID))
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	return &flowStoreImpl{
		db:        s.db,
		keyPrefix: flowStoreKeyPrefix(clusterID),
	}
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(clusterID string) (store.FlowStore, error) {
	fs := &flowStoreImpl{
		db:        s.db,
		keyPrefix: flowStoreKeyPrefix(clusterID),
	}
	return fs, nil
}

// RemoveFlowStore deletes the bucket holding the flow information for the graph in that cluster.
func (s *clusterStoreImpl) RemoveFlowStore(clusterID string) error {
	keyPrefix := flowStoreKeyPrefix(clusterID)
	return badgerhelper.RetryableUpdate(s.db, func(txn *badger.Txn) error {
		return badgerhelper.DeletePrefixRange(txn, keyPrefix)
	})
}
