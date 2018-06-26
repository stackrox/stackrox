package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const serviceIdentityBucket = "service_identities"

// Store provides storage functionality for alerts.
type Store interface {
	GetServiceIdentities() ([]*v1.ServiceIdentity, error)
	AddServiceIdentity(identity *v1.ServiceIdentity) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, serviceIdentityBucket)
	return &storeImpl{
		DB: db,
	}
}
