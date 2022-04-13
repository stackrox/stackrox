package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var serviceIdentityBucket = []byte("service_identities")

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	GetServiceIdentities() ([]*storage.ServiceIdentity, error)
	AddServiceIdentity(identity *storage.ServiceIdentity) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, serviceIdentityBucket)
	return &storeImpl{
		DB: db,
	}
}
