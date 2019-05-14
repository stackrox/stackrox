package globaldatastore

import (
	"context"

	"github.com/stackrox/rox/central/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper GlobalDataStore

// GlobalDataStore is the global datastore for all nodes across all clusters.
type GlobalDataStore interface {
	GetAllClusterNodeStores(ctx context.Context, writeAccess bool) (map[string]datastore.DataStore, error)
	GetClusterNodeStore(ctx context.Context, clusterID string, writeAccess bool) (datastore.DataStore, error)
	RemoveClusterNodeStores(ctx context.Context, clusterIDs ...string) error

	CountAllNodes(ctx context.Context) (int, error)

	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}
