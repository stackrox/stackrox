// This file was originally generated with
// //go:generate cp ../../../../central/policy/store/boltdb/store.go  .

package legacy

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var (
	policyBucket               = []byte("policies")
	removedDefaultPolicyBucket = []byte("removed_default_policies")
	policyCtx                  = context.Background()
	log                        = loghelper.LogWrapper{}
)

// Store provides storage functionality for policies.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.Policy, error)
	Upsert(ctx context.Context, policy *storage.Policy) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	bolthelper.RegisterBucketOrPanic(db, removedDefaultPolicyBucket)
	s := &storeImpl{
		DB: db,
	}
	return s
}
