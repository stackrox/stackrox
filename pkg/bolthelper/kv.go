package bolthelper

import "github.com/boltdb/bolt"

// KV is a key/value pair.
type KV struct {
	Key   []byte
	Value []byte
}

// PutAll inserts the given key/value pairs into the DB. Its main use case is to reduce the time the write lock is held
// for bulk upserts, by moving serialization outside of the transaction.
func PutAll(b *bolt.Bucket, kvs ...KV) error {
	for _, kv := range kvs {
		if err := b.Put(kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return nil
}
