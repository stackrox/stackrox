package globalstore

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper GlobalStore

// GlobalStore stores the nodes for all clusters.
type GlobalStore interface {
	GetClusterNodeStore(clusterID string) (store.Store, error)
	RemoveClusterNodeStore(clusterID string) error

	CountAllNodes() (int, error)
	Search(q *v1.Query) ([]search.Result, error)
}

// NewGlobalStore returns a new global node store for the given Bolt DB.
func NewGlobalStore(db *bolt.DB, indexer index.Indexer) (GlobalStore, error) {
	bolthelper.RegisterBucketOrPanic(db, nodesBucketKey)
	gsi := &globalStoreImpl{
		bucketRef: bolthelper.TopLevelRef(db, nodesBucketKey),
		indexer:   indexer,
	}
	if err := gsi.buildIndex(); err != nil {
		return nil, err
	}
	return gsi, nil
}
