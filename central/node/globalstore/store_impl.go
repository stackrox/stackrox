package globalstore

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	"github.com/stackrox/rox/pkg/search"
)

var (
	nodesBucketKey = []byte("nodes")
)

type globalStoreImpl struct {
	bucketRef bolthelper.BucketRef

	indexer index.Indexer
}

func alloc() proto.Message {
	return new(storage.Node)
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.Node).GetId())
}

func (s *globalStoreImpl) buildIndex() error {
	var childBuckets []string
	err := s.bucketRef.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, _ []byte) error {
			childBuckets = append(childBuckets, string(k))
			return nil
		})
	})
	if err != nil {
		return err
	}

	for _, k := range childBuckets {
		nodeStore, err := s.GetClusterNodeStore(k)
		if err != nil {
			return err
		}
		nodes, err := nodeStore.ListNodes()
		if err != nil {
			return err
		}
		if err := s.indexer.AddNodes(nodes); err != nil {
			return err
		}
	}
	return nil
}

func (s *globalStoreImpl) GetClusterNodeStore(clusterID string) (store.Store, error) {
	err := s.bucketRef.Update(func(b *bolt.Bucket) error {
		_, err := b.CreateBucketIfNotExists([]byte(clusterID))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not create per-cluster bucket: %v", err)
	}
	crud := protoCrud.NewMessageCrudForBucket(bolthelper.NestedRef(s.bucketRef, []byte(clusterID)), key, alloc)
	return datastore.New(store.New(crud), s.indexer), nil
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

// Search returns any node matches to the query
func (s *globalStoreImpl) Search(q *v1.Query) ([]search.Result, error) {
	return s.indexer.Search(q)
}
