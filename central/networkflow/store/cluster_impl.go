package store

import (
	"github.com/stackrox/rox/pkg/bolthelper"

	"github.com/boltdb/bolt"
)

type clusterStoreImpl struct {
	clusterFlowsBucket bolthelper.BucketRef
}

// GetAllFlowStores returns all of the FlowStores that exists for all clusters.
func (s *clusterStoreImpl) GetAllFlowStores() (flowStores []FlowStore) {
	s.clusterFlowsBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			flowStores = append(flowStores, s.wrapFlowStore(k))
			return nil
		})
	})
	return
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) FlowStore {
	return s.getFlowStore([]byte(clusterID))
}

func (s *clusterStoreImpl) getFlowStore(key []byte) (flowStore FlowStore) {
	s.clusterFlowsBucket.View(func(b *bolt.Bucket) error {
		if flowBucket := b.Bucket(key); flowBucket != nil {
			flowStore = s.wrapFlowStore(key)
		}
		return nil
	})
	return
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(clusterID string) (FlowStore, error) {
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

func (s *clusterStoreImpl) wrapFlowStore(key []byte) FlowStore {
	return &flowStoreImpl{
		flowsBucket: bolthelper.NestedRef(s.clusterFlowsBucket, key),
	}
}
