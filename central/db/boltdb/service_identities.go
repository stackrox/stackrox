package boltdb

import (
	"strconv"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const serviceIdentityBucket = "service_identities"

// GetServiceIdentities retrieves serviceIdentities from Bolt.
func (b *BoltDB) GetServiceIdentities() ([]*v1.ServiceIdentity, error) {
	var serviceIdentities []*v1.ServiceIdentity
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(serviceIdentityBucket))
		b.ForEach(func(k, v []byte) error {
			var serviceIdentity v1.ServiceIdentity
			if err := proto.Unmarshal(v, &serviceIdentity); err != nil {
				return err
			}
			serviceIdentities = append(serviceIdentities, &serviceIdentity)
			return nil
		})
		return nil
	})
	return serviceIdentities, err
}

func (b *BoltDB) upsertServiceIdentity(serviceIdentity *v1.ServiceIdentity) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(serviceIdentityBucket))
		bytes, err := proto.Marshal(serviceIdentity)
		if err != nil {
			return err
		}
		err = b.Put(serviceIdentityKey(serviceIdentity.Serial), bytes)
		return err
	})
}

// AddServiceIdentity adds a serviceIdentity to bolt
func (b *BoltDB) AddServiceIdentity(serviceIdentity *v1.ServiceIdentity) error {
	return b.upsertServiceIdentity(serviceIdentity)
}

func serviceIdentityKey(serial int64) []byte {
	return []byte(strconv.FormatInt(serial, 10))
}
