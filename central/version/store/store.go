package store

import (
	pgStore "github.com/stackrox/rox/central/version/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/rocksdb"
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

// NewPostgres returns a new postgres-based version store
func NewPostgres(pg postgres.DB) Store {
	return &storeImpl{pgStore: pgStore.New(pg)}
}
