package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const registryBucket = "registries"

func (b *BoltDB) getRegistry(id string, bucket *bolt.Bucket) (registry *v1.Registry, exists bool, err error) {
	registry = new(v1.Registry)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, registry)
	return
}

// GetRegistry returns registry with given id.
func (b *BoltDB) GetRegistry(id string) (registry *v1.Registry, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(registryBucket))
		registry, exists, err = b.getRegistry(id, bucket)
		return err
	})
	return
}

// GetRegistries retrieves registries from bolt
func (b *BoltDB) GetRegistries(request *v1.GetRegistriesRequest) ([]*v1.Registry, error) {
	var registries []*v1.Registry
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(registryBucket))
		b.ForEach(func(k, v []byte) error {
			var registry v1.Registry
			if err := proto.Unmarshal(v, &registry); err != nil {
				return err
			}
			registries = append(registries, &registry)
			return nil
		})
		return nil
	})
	return registries, err
}

// AddRegistry adds a registry into bolt
func (b *BoltDB) AddRegistry(registry *v1.Registry) (string, error) {
	registry.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(registryBucket))
		_, exists, err := b.getRegistry(registry.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Registry %v (%v) cannot be added because it already exists", registry.GetId(), registry.GetName())
		}
		bytes, err := proto.Marshal(registry)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(registry.GetId()), bytes)
	})
	return registry.Id, err
}

// UpdateRegistry upserts a registry into bolt
func (b *BoltDB) UpdateRegistry(registry *v1.Registry) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(registryBucket))
		bytes, err := proto.Marshal(registry)
		if err != nil {
			return err
		}
		return b.Put([]byte(registry.GetId()), bytes)
	})
}

// RemoveRegistry removes a registry from bolt
func (b *BoltDB) RemoveRegistry(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(registryBucket))
		return b.Delete([]byte(id))
	})
}
