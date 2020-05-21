package bolt

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var namespaceBucket = []byte("namespaces")

type storeImpl struct {
	*bolt.DB
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) store.Store {
	bolthelper.RegisterBucketOrPanic(db, namespaceBucket)
	return &storeImpl{
		DB: db,
	}
}

// GetNamespace returns namespace with given id.
func (b *storeImpl) Get(id string) (namespace *storage.NamespaceMetadata, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Namespace")
	namespace = new(storage.NamespaceMetadata)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(namespaceBucket)
		val := b.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, namespace)
	})

	return
}

// GetNamespaces retrieves namespaces matching the request from bolt
func (b *storeImpl) Walk(fn func(namespace *storage.NamespaceMetadata) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Namespace")
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(namespaceBucket)
		return b.ForEach(func(k, v []byte) error {
			var namespace storage.NamespaceMetadata
			if err := proto.Unmarshal(v, &namespace); err != nil {
				return err
			}
			return fn(&namespace)
		})
	})
	return err
}

func (b *storeImpl) Upsert(namespace *storage.NamespaceMetadata) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Namespace")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(namespaceBucket)
		bytes, err := proto.Marshal(namespace)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(namespace.GetId()), bytes)
	})
}

func (b *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Namespace")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(namespaceBucket)
		return bucket.Delete([]byte(id))
	})
}
