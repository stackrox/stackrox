package store

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	versionBucket = []byte("version")
)

// A Store stores versions.
type Store interface {
	// GetVersion returns the version found in the DB.
	// If there is no version in the DB, it returns nil and no error, so
	// the caller MUST always check for a nil return value.
	GetVersion() (*storage.Version, error)
	UpdateVersion(*storage.Version) error
}

// New returns a new ready-to-use store.
func New(boltDB *bolt.DB, badgerDB *badger.DB) Store {
	bolthelper.RegisterBucketOrPanic(boltDB, versionBucket)
	return &storeImpl{bucketRef: bolthelper.TopLevelRef(boltDB, versionBucket), badgerDB: badgerDB}
}
