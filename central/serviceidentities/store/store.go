package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const serviceIdentityBucket = "service_identities"

// Store provides storage functionality for alerts.
type Store interface {
	GetServiceIdentities() ([]*v1.ServiceIdentity, error)
	AddServiceIdentity(identity *v1.ServiceIdentity) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, serviceIdentityBucket)
	return &storeImpl{
		DB: db,
	}
}
