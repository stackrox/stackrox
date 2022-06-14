package testhelpers

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/migrator/bolthelpers"
	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
)

// MustGetObject populates the given object from the DB into the passed (allocated message).
func MustGetObject(t *testing.T, db *bolt.DB, bucketName []byte, id string, allocFunc func() proto.Message) proto.Message {
	bucketRef := bolthelpers.TopLevelRef(db, bucketName)
	var msg proto.Message
	assert.NoError(t, bucketRef.View(func(b *bolt.Bucket) error {
		bytes := b.Get([]byte(id))
		if bytes == nil {
			return nil
		}
		msg = allocFunc()
		err := proto.Unmarshal(bytes, msg)
		if err != nil {
			return err
		}
		return nil
	}))
	return msg
}

// MustInsertObject marshals and inserts the given object into the DB.
func MustInsertObject(t *testing.T, db *bolt.DB, bucketName []byte, id string, obj proto.Message) {
	bucketRef := bolthelpers.TopLevelRef(db, bucketName)
	assert.NoError(t, bucketRef.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(obj)
		if err != nil {
			return err
		}
		err = b.Put([]byte(id), bytes)
		if err != nil {
			return err
		}
		return nil
	}))
}
