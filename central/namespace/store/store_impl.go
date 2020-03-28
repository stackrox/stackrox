package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

// GetNamespace returns namespace with given id.
func (b *storeImpl) GetNamespace(id string) (namespace *storage.NamespaceMetadata, exists bool, err error) {
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
func (b *storeImpl) GetNamespaces() ([]*storage.NamespaceMetadata, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Namespace")
	var namespaces []*storage.NamespaceMetadata
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(namespaceBucket)
		return b.ForEach(func(k, v []byte) error {
			var namespace storage.NamespaceMetadata
			if err := proto.Unmarshal(v, &namespace); err != nil {
				return err
			}
			namespaces = append(namespaces, &namespace)
			return nil
		})
	})
	return namespaces, err
}

// AddNamespace adds a namespace to bolt
func (b *storeImpl) AddNamespace(namespace *storage.NamespaceMetadata) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Namespace")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(namespaceBucket)
		bytes, err := proto.Marshal(namespace)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(namespace.GetId()), bytes)
	})
}

// UpdateNamespace updates a namespace to bolt
func (b *storeImpl) UpdateNamespace(namespace *storage.NamespaceMetadata) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Namespace")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(namespaceBucket)
		bytes, err := proto.Marshal(namespace)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(namespace.GetId()), bytes)
	})
}

// RemoveNamespace removes a namespace.
func (b *storeImpl) RemoveNamespace(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Namespace")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(namespaceBucket)
		return bucket.Delete([]byte(id))
	})
}
