package bolt

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	ops "github.com/stackrox/stackrox/pkg/metrics"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB
}

// Get returns network policy with given id.
func (b *storeImpl) Get(_ context.Context, id string) (np *storage.NetworkPolicy, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkPolicy")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		np = new(storage.NetworkPolicy)
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, np)
	})
	return
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "NetworkPolicy")

	bytes, err := proto.Marshal(np)
	if err != nil {
		return err
	}
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)

		return bucket.Put([]byte(np.GetId()), bytes)
	})
}

// Delete removes a network policy
func (b *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkPolicy")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		return bucket.Delete([]byte(id))
	})
}
