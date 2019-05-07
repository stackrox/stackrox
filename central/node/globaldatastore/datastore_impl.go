package globaldatastore

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/node/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type globalDataStore struct {
	globalStore globalstore.GlobalStore

	indexer index.Indexer
}

// New creates and returns a new GlobalDataStore.
func New(globalStore globalstore.GlobalStore, indexer index.Indexer) (GlobalDataStore, error) {
	gds := &globalDataStore{
		globalStore: globalStore,
		indexer:     indexer,
	}
	if err := gds.buildIndex(); err != nil {
		return nil, err
	}
	return gds, nil
}

func (s *globalDataStore) buildIndex() error {
	nodeStores, err := s.globalStore.GetAllClusterNodeStores()
	if err != nil {
		return err
	}

	for _, nodeStore := range nodeStores {
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

func (s *globalDataStore) GetAllClusterNodeStores(ctx context.Context) (map[string]datastore.DataStore, error) {
	stores, err := s.globalStore.GetAllClusterNodeStores()
	if err != nil {
		return nil, err
	}

	dataStores := make(map[string]datastore.DataStore, len(stores))
	for clusterID, store := range stores {
		dataStores[clusterID] = datastore.New(store, s.indexer)
	}

	return dataStores, nil
}

func (s *globalDataStore) GetClusterNodeStore(ctx context.Context, clusterID string) (datastore.DataStore, error) {
	return s.globalStore.GetClusterNodeStore(clusterID)
}

func (s *globalDataStore) RemoveClusterNodeStores(ctx context.Context, clusterIDs ...string) error {
	return s.globalStore.RemoveClusterNodeStores(clusterIDs...)
}

func (s *globalDataStore) CountAllNodes(ctx context.Context) (int, error) {
	return s.globalStore.CountAllNodes()
}

// SearchResults returns any node matches to the query
func (s *globalDataStore) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	stores, err := s.GetAllClusterNodeStores(ctx)
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
func (s *globalDataStore) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.indexer.Search(q)
}
