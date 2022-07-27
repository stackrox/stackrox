// This file was originally generated with
// //go:generate cp ../../../../central/networkpolicies/datastore/internal/store/bolt/store.go .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var networkPolicyBucket = []byte("networkpolicies")

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) *storeImpl {
	bolthelper.RegisterBucketOrPanic(db, networkPolicyBucket)
	return &storeImpl{
		DB: db,
	}
}

// Store provides the interface to the underlying storage
type Store interface {
	Upsert(ctx context.Context, obj *storage.NetworkPolicy) error
	Walk(ctx context.Context, fn func(obj *storage.NetworkPolicy) error) error
}
