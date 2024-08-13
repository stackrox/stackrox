package suppress

import (
	"context"

	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// NodeCVEUnsnoozer provides an interface to search and unsnooze NodeCVEs including orphaned CVEs
type NodeCVEUnsnoozer struct {
	datastore nodeCVEDataStore.DataStore
}

// NewNodeCVEUnsnoozer returns a new instance of NodeCVEUnsnoozer
func NewNodeCVEUnsnoozer(datastore nodeCVEDataStore.DataStore) *NodeCVEUnsnoozer {
	return &NodeCVEUnsnoozer{
		datastore: datastore,
	}
}

// Search searches Node CVEs (including orphaned CVEs) that match the given query
func (u *NodeCVEUnsnoozer) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return u.datastore.Search(ctx, q, true)
}

// Unsuppress unsnoozes given Node CVEs
func (u *NodeCVEUnsnoozer) Unsuppress(ctx context.Context, cves ...string) error {
	return u.datastore.Unsuppress(ctx, cves...)
}
