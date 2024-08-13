package searchers

import (
	"context"

	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// NodeCVESearcher provides an interface to search for unorphaned Node CVEs
type NodeCVESearcher struct {
	datastore nodeCVEDataStore.DataStore
}

// NewNodeCVESearcher returns a new instance of NodeCVESearcher
func NewNodeCVESearcher(datastore nodeCVEDataStore.DataStore) *NodeCVESearcher {
	return &NodeCVESearcher{
		datastore: datastore,
	}
}

// Search searches unorphaned Node CVEs that match the query
func (s *NodeCVESearcher) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return s.datastore.Search(ctx, q, false)
}

// Count returns the number of unorphaned Node CVEs that match the query
func (s *NodeCVESearcher) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.datastore.Count(ctx, q, false)
}

// SearchNodeCVEs searches unorphaned Node CVEs that match the query
func (s *NodeCVESearcher) SearchNodeCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.datastore.SearchNodeCVEs(ctx, q, false)
}
