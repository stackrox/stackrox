package bolthelper

import (
	bolt "go.etcd.io/bbolt"
)

// Exists checks if the key exists in the bucket
func Exists(b *bolt.Bucket, id string) bool {
	return ExistsBytes(b, []byte(id))
}

// ExistsBytes checks if they key (passed as []byte) exists in the bucket
func ExistsBytes(b *bolt.Bucket, id []byte) bool {
	return b.Get(id) != nil
}
