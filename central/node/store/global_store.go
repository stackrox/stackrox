package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/pkg/bolthelper"
)

// GlobalStore stores the nodes for all clusters.
type GlobalStore interface {
	GetClusterNodeStore(clusterID string) (Store, error)

	CountAllNodes() (int, error)
}

// NewGlobalStore returns a new global node store for the given Bolt DB.
func NewGlobalStore(db *bolt.DB) GlobalStore {
	bolthelper.RegisterBucketOrPanic(db, nodesBucketKey)
	return &globalStoreImpl{
		bucketRef: bolthelper.TopLevelRef(db, []byte(nodesBucketKey)),
	}
}
