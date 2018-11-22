package secondarykey

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
)

// GetCurrentUniqueKey returns the secondary key for the input primary key.
func GetCurrentUniqueKey(tx *bolt.Tx, bucket string, id string) (string, bool) {
	b := tx.Bucket(getMapperBucket(bucket))
	val := b.Get([]byte(id))
	if val == nil {
		return "", false
	}
	return string(val), true
}

// CheckUniqueKeyExistsAndInsert checks if the name exists within the context of a transaction which means
// if the transaction fails then this will be rolled back
func CheckUniqueKeyExistsAndInsert(tx *bolt.Tx, bucket string, id, k string) error {
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

// InsertUniqueKey inserts the unique key
func InsertUniqueKey(tx *bolt.Tx, bucket string, id, k string) error {
	b := tx.Bucket(getUniqueBucket(bucket))
	if err := b.Put([]byte(k), []byte{}); err != nil {
		return err
	}
	b = tx.Bucket(getMapperBucket(bucket))
	return b.Put([]byte(id), []byte(k))
}

// UpdateUniqueKey changes a current key to a new value.
func UpdateUniqueKey(tx *bolt.Tx, bucket string, id, k string) error {
	if _, exists := GetCurrentUniqueKey(tx, bucket, id); exists {
		RemoveUniqueKey(tx, bucket, id)
	}
	return CheckUniqueKeyExistsAndInsert(tx, bucket, id, k)
}

// RemoveUniqueKey removes a secondary key.
func RemoveUniqueKey(tx *bolt.Tx, bucket string, id string) error {
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

// CreateUniqueKeyBucket creates buckets for storing secondary keys and their mappings to primary keys.
func CreateUniqueKeyBucket(tx *bolt.Tx, bucket string) error {
	if _, err := tx.CreateBucketIfNotExists(getUniqueBucket(bucket)); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists(getMapperBucket(bucket))
	return err
}

func getUniqueBucket(b string) []byte {
	return []byte(b + "-unique")
}

func getMapperBucket(b string) []byte {
	return []byte(b + "-mapper")
}
