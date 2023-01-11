package globaldatastore

import (
	"context"

	"github.com/stackrox/rox/central/node/datastore"
)

// GlobalDataStore is the global datastore for all nodes across all clusters.
//
//go:generate mockgen-wrapper
type GlobalDataStore interface {
	GetAllClusterNodeStores(ctx context.Context, writeAccess bool) (map[string]datastore.DataStore, error)
	RemoveClusterNodeStores(ctx context.Context, clusterIDs ...string) error
}
