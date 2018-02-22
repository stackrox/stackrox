package boltdb

import (
	"fmt"

	"github.com/boltdb/bolt"
)

func getUniqueBucket(b string) []byte {
	return []byte(b + "-unique")
}

func getMapperBucket(b string) []byte {
	return []byte(b + "-mapper")
}

func getCurrentUniqueKey(tx *bolt.Tx, bucket string, id string) string {
	b := tx.Bucket(getMapperBucket(bucket))
	val := b.Get([]byte(id))
	if val == nil {
		return ""
	}
	return string(val)
}

// checks if the name exists within the context of a transaction which means if the transaction fails then this
// will be rolled back
func checkUniqueKeyExistsAndInsert(tx *bolt.Tx, bucket string, id, k string) error {
	b := tx.Bucket(getUniqueBucket(bucket))
	if b.Get([]byte(k)) != nil {
		return fmt.Errorf("'%v' already exists", k)
	}
	if err := b.Put([]byte(k), []byte{}); err != nil {
		return err
	}
	b = tx.Bucket(getMapperBucket(bucket))
	return b.Put([]byte(id), []byte(k))
}

func createUniqueKeyBucket(tx *bolt.Tx, bucket string) error {
	if _, err := tx.CreateBucketIfNotExists(getUniqueBucket(bucket)); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists(getMapperBucket(bucket))
	return err
}

func removeUniqueKey(tx *bolt.Tx, bucket string, id string) error {
	b := tx.Bucket(getMapperBucket(bucket))
	val := b.Get([]byte(id))
	if val == nil {
		return fmt.Errorf("Could not remove %v because it does not exist", id)
	}
	if err := b.Delete([]byte(id)); err != nil {
		return err
	}
	b = tx.Bucket(getUniqueBucket(bucket))
	return b.Delete(val)
}
