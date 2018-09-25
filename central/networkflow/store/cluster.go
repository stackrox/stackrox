package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const clusterFlowBucket = "clustersWithFlowsBucket"

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetAllFlowStores() []FlowStore
	GetFlowStore(clusterID string) FlowStore

	CreateFlowStore(clusterID string) FlowStore
	RemoveFlowStore(clusterID string) error
}

// NewClusterStore returns a new ClusterStore instance using the provided bolt DB instance.
func NewClusterStore(db *bolt.DB) ClusterStore {
	bolthelper.RegisterBucketOrPanic(db, clusterFlowBucket)
	return &clusterStoreImpl{
		db: db,
	}
}
