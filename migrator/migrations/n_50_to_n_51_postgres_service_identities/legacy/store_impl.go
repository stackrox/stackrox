// This file was originally generated with
// //go:generate cp ../../../../central/serviceidentities/internal/store/bolt/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var serviceIdentityBucket = []byte("service_identities")

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) *storeImpl {
	bolthelper.RegisterBucketOrPanic(db, serviceIdentityBucket)
	return &storeImpl{
		DB: db,
	}
}

type storeImpl struct {
	*bolt.DB
}

// GetAll retrieves serviceIdentities from Bolt.
func (b *storeImpl) GetAll(_ context.Context) ([]*storage.ServiceIdentity, error) {
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
		err = b.Put([]byte(serviceIdentity.GetSerialStr()), bytes)
		return err
	})
}

// Upsert adds a serviceIdentity to bolt
func (b *storeImpl) Upsert(_ context.Context, serviceIdentity *storage.ServiceIdentity) error {
	return b.upsertServiceIdentity(serviceIdentity)
}
