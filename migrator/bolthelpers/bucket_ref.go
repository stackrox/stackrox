package bolthelpers

import (
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// BucketRef is a reference to a bucket. The user does not need to care whether this is a top-level bucket, or a nested
// bucket. However, the user needs to ensure that it exists - no facilities are provided as part of this interface to
// create or destroy a referenced bucket.
type BucketRef interface {
	View(func(b *bolt.Bucket) error) error
	Update(func(b *bolt.Bucket) error) error
}

// TopLevelRef obtains a BucketRef for a top-level bucket in the DB.
func TopLevelRef(db *bolt.DB, key []byte) BucketRef {
	return &topLevelBucketRef{
		db:  db,
		key: key,
	}
}

type topLevelBucketRef struct {
	db  *bolt.DB
	key []byte
}

func (r *topLevelBucketRef) getApplyFunc(fn func(b *bolt.Bucket) error) func(tx *bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		bucket := tx.Bucket(r.key)
		if bucket == nil {
			return fmt.Errorf("no such bucket: %v", r.key)
		}
		return fn(bucket)
	}
}

func (r *topLevelBucketRef) View(fn func(b *bolt.Bucket) error) error {
	return r.db.View(r.getApplyFunc(fn))
}

func (r *topLevelBucketRef) Update(fn func(b *bolt.Bucket) error) error {
	return r.db.Update(r.getApplyFunc(fn))
}
