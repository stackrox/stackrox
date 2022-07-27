// This file was originally generated with
// //go:generate cp ../../../../central/networkpolicies/datastore/internal/store/bolt/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) Walk(_ context.Context, fn func(np *storage.NetworkPolicy) error) error {
	return b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var np storage.NetworkPolicy
			if err := proto.Unmarshal(v, &np); err != nil {
				return err
			}
			return fn(&np)
		})
	})
}

// Upsert upserts a network policy to bolt
func (b *storeImpl) Upsert(_ context.Context, np *storage.NetworkPolicy) error {
	bytes, err := proto.Marshal(np)
	if err != nil {
		return err
	}
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)

		return bucket.Put([]byte(np.GetId()), bytes)
	})
}
