package store

import (
	"strconv"
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

// GetServiceIdentities retrieves serviceIdentities from Bolt.
func (b *storeImpl) GetServiceIdentities() ([]*storage.ServiceIdentity, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ServiceIdentity")
	var serviceIdentities []*storage.ServiceIdentity
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(serviceIdentityBucket))
		return b.ForEach(func(k, v []byte) error {
			var serviceIdentity storage.ServiceIdentity
			if err := proto.Unmarshal(v, &serviceIdentity); err != nil {
				return err
			}
			serviceIdentities = append(serviceIdentities, &serviceIdentity)
			return nil
		})
	})
	return serviceIdentities, err
}

func (b *storeImpl) upsertServiceIdentity(serviceIdentity *storage.ServiceIdentity) error {
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
func (b *storeImpl) AddServiceIdentity(serviceIdentity *storage.ServiceIdentity) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "ServiceIdentity")
	return b.upsertServiceIdentity(serviceIdentity)
}

func serviceIdentityKey(serial int64) []byte {
	return []byte(strconv.FormatInt(serial, 10))
}
