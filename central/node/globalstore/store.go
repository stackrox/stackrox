package globalstore

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/pkg/bolthelper"
)

// GlobalStore stores the nodes for all clusters.
type GlobalStore interface {
	GetAllClusterNodeStores() (map[string]store.Store, error)
	GetClusterNodeStore(clusterID string, writeAccess bool) (store.Store, error)
	RemoveClusterNodeStores(clusterIDs ...string) error

	CountAllNodes() (int, error)
}

// NewGlobalStore returns a new global node store for the given Bolt DB.
func NewGlobalStore(db *bolt.DB) GlobalStore {
	bolthelper.RegisterBucketOrPanic(db, nodesBucketKey)
	gsi := &globalStoreImpl{
		bucketRef: bolthelper.TopLevelRef(db, nodesBucketKey),
	}
	return gsi
}

//go:generate mockgen-wrapper GlobalStore
