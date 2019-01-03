package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/bolthelper"
)

//go:generate mockgen-wrapper GlobalStore

// GlobalStore stores the nodes for all clusters.
type GlobalStore interface {
	GetClusterNodeStore(clusterID string) (Store, error)
	RemoveClusterNodeStore(clusterID string) error

	CountAllNodes() (int, error)
}

// NewGlobalStore returns a new global node store for the given Bolt DB.
func NewGlobalStore(db *bolt.DB) GlobalStore {
	bolthelper.RegisterBucketOrPanic(db, nodesBucketKey)
	return &globalStoreImpl{
		bucketRef: bolthelper.TopLevelRef(db, nodesBucketKey),
	}
}
