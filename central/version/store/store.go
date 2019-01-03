package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	versionBucket = []byte("version")
)

// A ReadOnlyStore is a read-only snapshot of the version store.
type ReadOnlyStore interface {
	GetVersion() (*storage.Version, error)
}

// A Store stores versions.
type Store interface {
	ReadOnlyStore
	UpdateVersion(*storage.Version) error
}

// New returns a new ready-to-use store.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, versionBucket)
	return &storeImpl{bucketRef: bolthelper.TopLevelRef(db, versionBucket)}
}
