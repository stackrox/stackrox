package store

import (
	"fmt"

	"github.com/boltdb/bolt"
)

type clusterStoreImpl struct {
	db *bolt.DB
}

// GetAllFlowStores returns all of the FlowStores that exists for all clusters.
func (s *clusterStoreImpl) GetAllFlowStores() (flowStores []FlowStore) {
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterFlowBucket))
		return b.ForEach(func(k, v []byte) error {
			flowStores = append(flowStores, s.wrapFlowStore(string(k)))
			return nil
		})
	})
	return
}

// GetFlowStore returns the FlowStore for the cluster ID, or nil if none exists.
func (s *clusterStoreImpl) GetFlowStore(clusterID string) (flowStore FlowStore) {
	s.db.Update(func(tx *bolt.Tx) error {
		if hasFlowStoreForCluster(tx, clusterID) {
			flowStore = s.wrapFlowStore(clusterID)
		}
		return nil
	})
	return
}

// CreateFlowStore returns the FlowStore for the cluster ID, or creates one if none exists.
func (s *clusterStoreImpl) CreateFlowStore(clusterID string) (flowStore FlowStore) {
	flowStore = s.GetFlowStore(clusterID)
	if flowStore == nil {
		s.db.Update(func(tx *bolt.Tx) error {
			registerFlowStore(tx, clusterID)
			return nil
		})
		flowStore = NewFlowStore(s.db, clusterID)
	}
	return
}

// RemoveFlowStore deletes the bucket holding the flow information for the graph in that cluster.
func (s *clusterStoreImpl) RemoveFlowStore(clusterID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		unregisterFlowStore(tx, clusterID)
		return deleteFlowStore(tx, clusterID)
	})
}

// Member helper functions.
///////////////////////////

func (s *clusterStoreImpl) wrapFlowStore(clusterID string) FlowStore {
	return &flowStoreImpl{
		db:         s.db,
		bucketName: clusterID,
	}
}

// Static helper functions.
////////////////////////////

func hasFlowStoreForCluster(tx *bolt.Tx, clusterID string) (hasStore bool) {
	bucket := tx.Bucket([]byte(clusterFlowBucket))
	store := bucket.Get([]byte(clusterID))
	if store != nil {
		hasStore = true
	}
	return
}

func registerFlowStore(tx *bolt.Tx, clusterID string) {
	bucket := tx.Bucket([]byte(clusterFlowBucket))
	bSlice := []byte(clusterID)
	bucket.Put(bSlice, bSlice)
}

func unregisterFlowStore(tx *bolt.Tx, clusterID string) {
	b := tx.Bucket([]byte(clusterFlowBucket))
	b.Delete([]byte(clusterID))
}

func deleteFlowStore(tx *bolt.Tx, clusterID string) error {
	flowBucketName := networkFlowBucket + clusterID
	if flowBucket := tx.Bucket([]byte(flowBucketName)); flowBucket != nil {
		if err := tx.DeleteBucket([]byte(flowBucketName)); err != nil {
			return fmt.Errorf("unable to delete flow bucket: %s", err)
		}
	}
	return nil
}
