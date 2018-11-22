package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	authProviderBucket  = "authProviders"
	authValidatedBucket = "authValidated"
)

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) authproviders.Store {
	bolthelper.RegisterBucketOrPanic(db, authProviderBucket)
	bolthelper.RegisterBucketOrPanic(db, authValidatedBucket)
	return &storeImpl{
		DB: db,
	}
}
