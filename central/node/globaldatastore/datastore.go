package globaldatastore

import (
	"github.com/stackrox/rox/central/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper GlobalDataStore

// GlobalDataStore is the global datastore for all nodes across all clusters.
type GlobalDataStore interface {
	GetAllClusterNodeStores() (map[string]datastore.DataStore, error)
	GetClusterNodeStore(clusterID string) (datastore.DataStore, error)
	RemoveClusterNodeStores(clusterIDs ...string) error

	CountAllNodes() (int, error)

	SearchResults(q *v1.Query) ([]*v1.SearchResult, error)
	Search(q *v1.Query) ([]search.Result, error)
}
