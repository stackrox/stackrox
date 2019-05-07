package globalstore

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

var (
	nodesBucketKey = []byte("nodes")
)

type globalStoreImpl struct {
	bucketRef bolthelper.BucketRef
}

func alloc() proto.Message {
	return new(storage.Node)
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.Node).GetId())
}

func (s *globalStoreImpl) GetAllClusterNodeStores() (map[string]store.Store, error) {
	stores := make(map[string]store.Store)
	err := s.bucketRef.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, _ []byte) error {
			crud := protoCrud.NewMessageCrudForBucket(bolthelper.NestedRef(s.bucketRef, k), key, alloc)
			stores[string(k)] = store.New(crud)
			return nil
		})
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not get all cluster nodes")
	}
	return stores, nil
}

func (s *globalStoreImpl) GetClusterNodeStore(clusterID string) (store.Store, error) {
	err := s.bucketRef.Update(func(b *bolt.Bucket) error {
		_, err := b.CreateBucketIfNotExists([]byte(clusterID))
		return err
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not create per-cluster bucket")
	}
	crud := protoCrud.NewMessageCrudForBucket(bolthelper.NestedRef(s.bucketRef, []byte(clusterID)), key, alloc)
	return store.New(crud), nil
}

func (s *globalStoreImpl) RemoveClusterNodeStores(clusterIDs ...string) error {
	if len(clusterIDs) == 0 {
		return nil
	}
	return s.bucketRef.Update(func(b *bolt.Bucket) error {
		for _, clusterID := range clusterIDs {
			key := []byte(clusterID)
			if b.Bucket(key) != nil {
				if err := b.DeleteBucket(key); err != nil {
					return err
				}
			}
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
