package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const registryBucket = "registries"

// GetRegistry returns registry with given name.
func (b *BoltDB) GetRegistry(name string) (registry *v1.Registry, exists bool, err error) {
	registry = new(v1.Registry)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(registryBucket))
		val := b.Get([]byte(name))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, registry)
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

func (b *BoltDB) upsertRegistry(registry *v1.Registry) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(registryBucket))
		bytes, err := proto.Marshal(registry)
		if err != nil {
			return err
		}
		err = b.Put([]byte(registry.Name), bytes)
		return err
	})
}

// AddRegistry upserts a registry into bolt
func (b *BoltDB) AddRegistry(registry *v1.Registry) error {
	return b.upsertRegistry(registry)
}

// UpdateRegistry upserts a registry into bolt
func (b *BoltDB) UpdateRegistry(registry *v1.Registry) error {
	return b.upsertRegistry(registry)
}

// RemoveRegistry removes a registry from bolt
func (b *BoltDB) RemoveRegistry(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(registryBucket))
		err := b.Delete([]byte(name))
		return err
	})
}
