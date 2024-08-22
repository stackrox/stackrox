package searchers

import (
	"context"

	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// NonOrphanedNodeCVESearcher provides an interface to search for non-orphaned Node CVEs
type NonOrphanedNodeCVESearcher struct {
	datastore nodeCVEDataStore.DataStore
}

// NewNonOrphanedNodeCVESearcher returns a new instance of NodeCVESearcher
func NewNonOrphanedNodeCVESearcher(datastore nodeCVEDataStore.DataStore) *NonOrphanedNodeCVESearcher {
	return &NonOrphanedNodeCVESearcher{
		datastore: datastore,
	}
}

// Search searches unorphaned Node CVEs that match the query
func (s *NonOrphanedNodeCVESearcher) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	q = common.WithoutOrphanedNodeCVEsQuery(q)
	return s.datastore.Search(ctx, q)
}

// Count returns the number of unorphaned Node CVEs that match the query
func (s *NonOrphanedNodeCVESearcher) Count(ctx context.Context, q *v1.Query) (int, error) {
	q = common.WithoutOrphanedNodeCVEsQuery(q)
	return s.datastore.Count(ctx, q)
}

// SearchNodeCVEs searches unorphaned Node CVEs that match the query
func (s *NonOrphanedNodeCVESearcher) SearchNodeCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	q = common.WithoutOrphanedNodeCVEsQuery(q)
	return s.datastore.SearchNodeCVEs(ctx, q)
}
