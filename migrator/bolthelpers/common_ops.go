package bolthelpers

import (
	bolt "github.com/etcd-io/bbolt"
)

// RetrieveElementAtKey retrieves the element at the given key from the given BucketRef.
func RetrieveElementAtKey(bucketRef BucketRef, key []byte) ([]byte, error) {
	var val []byte
	err := bucketRef.View(func(b *bolt.Bucket) error {
		val = b.Get(key)
		return nil
	})
	return val, err
}
