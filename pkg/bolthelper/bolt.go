// This code is adapted from https://github.com/boltdb/bolt/blob/master/cmd/bolt/main.go
// which is licensed under the MIT License

package bolthelper

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// DBFileName is the name of the file (within `migrations.DBMountPath`) containing the Bolt database.
	DBFileName = "stackrox.db"
	txMaxSize  = 65536
)

// New returns an instance of the persistent BoltDB store
func New(path string) (*bolt.DB, error) {
	dirPath := filepath.Dir(path)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0600)
		if err != nil {
			return nil, errors.Wrapf(err, "Error creating db path %v", dirPath)
		}
	} else if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// NewWithDefaults returns an instance of the persistent BoltDB store with default values loaded.
func NewWithDefaults() (*bolt.DB, error) {
	db, err := New(filepath.Join(migrations.DBMountPath, DBFileName))
	if err != nil {
		return db, err
	}

	return db, nil
}

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(dbPath string) (*bolt.DB, error) {
	tmpDir, err := ioutil.TempDir("", "")
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

// Compact compacts the BoltDB from src to dst
func Compact(dst, src *bolt.DB) error {
	// commit regularly, or we'll run out of memory for large datasets if using one transaction.
	var size int64
	tx, err := dst.Begin(true)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(tx.Rollback)

	if err := walk(src, func(keys [][]byte, k, v []byte, seq uint64) error {
		// On each key/value, check if we have exceeded tx size.
		sz := int64(len(k) + len(v))
		if size+sz > txMaxSize {
			// Commit previous transaction.
			if err := tx.Commit(); err != nil {
				return err
			}

			// Start new transaction.
			tx, err = dst.Begin(true)
			if err != nil {
				return err
			}
			size = 0
		}
		size += sz

		// Create bucket on the root transaction if this is the first level.
		nk := len(keys)
		if nk == 0 {
			bkt, err := tx.CreateBucket(k)
			if err != nil {
				return err
			}
			return bkt.SetSequence(seq)
		}

		// Create buckets on subsequent levels, if necessary.
		b := tx.Bucket(keys[0])
		if nk > 1 {
			for _, k := range keys[1:] {
				b = b.Bucket(k)
			}
		}

		// If there is no value then this is a bucket call.
		if v == nil {
			bkt, err := b.CreateBucket(k)
			if err != nil {
				return err
			}
			return bkt.SetSequence(seq)
		}

		// Otherwise treat it as a key/value pair.
		return b.Put(k, v)
	}); err != nil {
		return err
	}

	return tx.Commit()
}

// walkFunc is the type of the function called for keys (buckets and "normal"
// values) discovered by Walk. keys is the list of keys to descend to the bucket
// owning the discovered key/value pair k/v.
type walkFunc func(keys [][]byte, k, v []byte, seq uint64) error

// walk walks recursively the bolt database db, calling walkFn for each key it finds.
func walk(db *bolt.DB, walkFn walkFunc) error {
	return db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			return walkBucket(b, nil, name, nil, b.Sequence(), walkFn)
		})
	})
}

func walkBucket(b *bolt.Bucket, keypath [][]byte, k, v []byte, seq uint64, fn walkFunc) error {
	// Execute callback.
	if err := fn(keypath, k, v, seq); err != nil {
		return err
	}

	// If this is not a bucket then stop.
	if v != nil {
		return nil
	}

	// Iterate over each child key/value.
	keypath = append(keypath, k)
	return b.ForEach(func(k, v []byte) error {
		if v == nil {
			bkt := b.Bucket(k)
			return walkBucket(bkt, keypath, k, nil, bkt.Sequence(), fn)
		}
		return walkBucket(b, keypath, k, v, b.Sequence(), fn)
	})
}
