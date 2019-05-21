package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	authProviderBucket = []byte("authProviders")
)

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) authproviders.Store {
	bolthelper.RegisterBucketOrPanic(db, authProviderBucket)
	return &storeImpl{
		DB: db,
	}
}
