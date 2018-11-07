package store

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

const (
	nodesBucketKey = "nodes"
)

type globalStoreImpl struct {
	bucketRef bolthelper.BucketRef
}

func alloc() proto.Message {
	return new(v1.Node)
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*v1.Node).GetId())
}

func (s *globalStoreImpl) GetClusterNodeStore(clusterID string) (Store, error) {
	err := s.bucketRef.Update(func(b *bolt.Bucket) error {
		_, err := b.CreateBucketIfNotExists([]byte(clusterID))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not create per-cluster bucket: %v", err)
	}
	crud := protoCrud.NewMessageCrudForBucket(bolthelper.NestedRef(s.bucketRef, []byte(clusterID)), key, alloc)
	return &storeImpl{crud: crud}, nil
}

func (s *globalStoreImpl) RemoveClusterNodeStore(clusterID string) error {
	key := []byte(clusterID)
	return s.bucketRef.Update(func(b *bolt.Bucket) error {
		if b.Bucket(key) != nil {
			return b.DeleteBucket(key)
		}
		return nil
	})
}

func (s *globalStoreImpl) CountAllNodes() (int, error) {
	numNodes := 0
	err := s.bucketRef.View(func(b *bolt.Bucket) error {
		return bolthelper.CountLeavesRecursive(b, -1, &numNodes)
	})
	if err != nil {
		return 0, err
	}
	return numNodes, nil
}
