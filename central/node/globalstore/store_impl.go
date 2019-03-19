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

func (s *globalStoreImpl) getAllClusterNodeStores() ([]store.Store, error) {
	var bytes [][]byte
	err := s.bucketRef.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, _ []byte) error {
			bytes = append(bytes, k)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("could not get all cluster nodes: %v", err)
	}
	stores := make([]store.Store, 0, len(bytes))
	for _, k := range bytes {
		crud := protoCrud.NewMessageCrudForBucket(bolthelper.NestedRef(s.bucketRef, k), key, alloc)
		stores = append(stores, datastore.New(store.New(crud), s.indexer))
	}
	return stores, nil
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

// SearchResults returns any node matches to the query
func (s *globalStoreImpl) SearchResults(q *v1.Query) ([]*v1.SearchResult, error) {
	stores, err := s.getAllClusterNodeStores()
	if err != nil {
		return nil, err
	}
	results, err := s.indexer.Search(q)
	if err != nil {
		return nil, err
	}

	searchResults := make([]*v1.SearchResult, 0, len(results))
	for _, r := range results {
		var node *storage.Node
		for _, s := range stores {
			node, err = s.GetNode(r.ID)
			if err == nil {
				break
			}
		}
		if node == nil {
			continue
		}
		searchResults = append(searchResults, &v1.SearchResult{
			Id:             r.ID,
			Name:           node.Name,
			Category:       v1.SearchCategory_NODES,
			FieldToMatches: search.GetProtoMatchesMap(r.Matches),
			Score:          r.Score,
			Location:       fmt.Sprintf("%s/%s", node.GetClusterName(), node.GetName()),
		})
	}
	return searchResults, nil
}

// Search returns any node matches to the query
func (s *globalStoreImpl) Search(q *v1.Query) ([]search.Result, error) {
	return s.indexer.Search(q)
}
