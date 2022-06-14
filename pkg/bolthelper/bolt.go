// This code is adapted from https://github.com/boltdb/bolt/blob/master/cmd/bolt/main.go
// which is licensed under the MIT License

package bolthelper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

const (
	// DBFileName is the name of the file (within `migrations.DBMountPath`) containing the Bolt database.
	DBFileName    = "stackrox.db"
	txMaxSize     = 65536
	dbOpenTimeout = 2 * time.Minute
)

// New returns an instance of the persistent BoltDB store
func New(path string) (*bolt.DB, error) {
	dirPath := filepath.Dir(path)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0700)
		if err != nil {
			return nil, errors.Wrapf(err, "Error creating db path %v", dirPath)
		}
	} else if err != nil {
		return nil, err
	}
	options := *bolt.DefaultOptions
	options.FreelistType = bolt.FreelistMapType
	options.Timeout = dbOpenTimeout
	db, err := bolt.Open(path, 0600, &options)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// NewWithDefaults returns an instance of the persistent BoltDB store with default values loaded.
func NewWithDefaults(dbPath string) (*bolt.DB, error) {
	return New(filepath.Join(dbPath, DBFileName))
}

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(dbPath string) (*bolt.DB, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}
	return New(filepath.Join(tmpDir, strings.Replace(dbPath, "/", "_", -1)))
}

// RegisterBucket registers a new bucket in the global DB.
func RegisterBucket(db *bolt.DB, bucket []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return errors.Wrap(err, "create bucket")
		}
		if err := secondarykey.CreateUniqueKeyBucket(tx, bucket); err != nil {
			return errors.Wrap(err, "create bucket")
		}
		return nil
	})
}

// RegisterBucketOrPanic registers a new bucket in the global DB, and panics if there's an error.
func RegisterBucketOrPanic(db *bolt.DB, bucket []byte) {
	err := RegisterBucket(db, bucket)
	if err != nil {
		panic(fmt.Sprintf("failed to register bucket %s: %s", bucket, err))
	}
}
