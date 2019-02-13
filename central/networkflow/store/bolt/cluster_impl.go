package bolt

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var clusterFlowBucket = []byte("clustersWithFlowsBucket")

// NewClusterStore returns a new ClusterStore instance using the provided bolt DB instance.
func NewClusterStore(db *bolt.DB) store.ClusterStore {
	bolthelper.RegisterBucketOrPanic(db, clusterFlowBucket)
	return &clusterStoreImpl{
		clusterFlowsBucket: bolthelper.TopLevelRef(db, clusterFlowBucket),
	}
}

type clusterStoreImpl struct {
	clusterFlowsBucket bolthelper.BucketRef
}

// GetAllFlowStores returns all of the FlowStores that exists for all clusters.
func (s *clusterStoreImpl) GetAllFlowStores() (flowStores []store.FlowStore) {
	s.clusterFlowsBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			flowStores = append(flowStores, s.wrapFlowStore(k))
			return nil
		})
	})
	return
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) store.FlowStore {
	return s.getFlowStore([]byte(clusterID))
}

func (s *clusterStoreImpl) getFlowStore(key []byte) (flowStore store.FlowStore) {
	s.clusterFlowsBucket.View(func(b *bolt.Bucket) error {
		if flowBucket := b.Bucket(key); flowBucket != nil {
			flowStore = s.wrapFlowStore(key)
		}
		return nil
	})
	return
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(clusterID string) (store.FlowStore, error) {
	key := []byte(clusterID)
	flowStore := s.getFlowStore(key)
	if flowStore != nil {
		return flowStore, nil
	}
	err := s.clusterFlowsBucket.Update(func(b *bolt.Bucket) error {
		_, err := b.CreateBucket(key)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.wrapFlowStore(key), nil
}

// RemoveFlowStore deletes the bucket holding the flow information for the graph in that cluster.
func (s *clusterStoreImpl) RemoveFlowStore(clusterID string) error {
	key := []byte(clusterID)
	return s.clusterFlowsBucket.Update(func(b *bolt.Bucket) error {
		return b.DeleteBucket(key)
	})
}

// Member helper functions.
///////////////////////////

func (s *clusterStoreImpl) wrapFlowStore(key []byte) store.FlowStore {
	return &flowStoreImpl{
		flowsBucket: bolthelper.NestedRef(s.clusterFlowsBucket, key),
	}
}
