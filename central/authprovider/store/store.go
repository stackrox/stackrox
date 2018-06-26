package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const authProviderBucket = "authProviders"

// Store provides storage functionality for alerts.
type Store interface {
	GetAuthProvider(id string) (*v1.AuthProvider, bool, error)
	GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error)
	AddAuthProvider(authProvider *v1.AuthProvider) (string, error)
	UpdateAuthProvider(authProvider *v1.AuthProvider) error
	RemoveAuthProvider(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, authProviderBucket)
	return &storeImpl{
		DB: db,
	}
}
