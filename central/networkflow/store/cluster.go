package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const clusterFlowBucket = "clustersWithFlowsBucket"

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetAllFlowStores() []FlowStore
	GetFlowStore(clusterID string) FlowStore

	CreateFlowStore(clusterID string) (FlowStore, error)
	RemoveFlowStore(clusterID string) error
}

// NewClusterStore returns a new ClusterStore instance using the provided bolt DB instance.
func NewClusterStore(db *bolt.DB) ClusterStore {
	bolthelper.RegisterBucketOrPanic(db, clusterFlowBucket)
	return &clusterStoreImpl{
		clusterFlowsBucket: bolthelper.TopLevelRef(db, []byte(clusterFlowBucket)),
	}
}
