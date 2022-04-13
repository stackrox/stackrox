package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	bolt "go.etcd.io/bbolt"
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
func New(boltDB *bolt.DB, rocksDB *rocksdb.RocksDB) Store {
	bolthelper.RegisterBucketOrPanic(boltDB, versionBucket)
	return &storeImpl{bucketRef: bolthelper.TopLevelRef(boltDB, versionBucket), rocksDB: rocksDB}
}
