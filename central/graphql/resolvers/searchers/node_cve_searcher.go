package searchers

import (
	"context"

	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type NodeCVESearcher struct {
	datastore nodeCVEDataStore.DataStore
}

func NewNodeCVESearcher(datastore nodeCVEDataStore.DataStore) *NodeCVESearcher {
	return &NodeCVESearcher{
		datastore: datastore,
	}
}

func (s *NodeCVESearcher) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return s.datastore.Search(ctx, q, false)
}

func (s *NodeCVESearcher) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.datastore.Count(ctx, q, false)
}

func (s *NodeCVESearcher) SearchNodeCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.datastore.SearchNodeCVEs(ctx, q, false)
}
