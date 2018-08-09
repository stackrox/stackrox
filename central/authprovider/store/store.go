package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	authProviderBucket  = "authProviders"
	authValidatedBucket = "authValidated"
)

// Store provides storage functionality for auth providers.
type Store interface {
	GetAuthProvider(id string) (*v1.AuthProvider, bool, error)
	GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error)
	AddAuthProvider(authProvider *v1.AuthProvider) (string, error)
	UpdateAuthProvider(authProvider *v1.AuthProvider) error
	RemoveAuthProvider(id string) error
	RecordAuthSuccess(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, authProviderBucket)
	bolthelper.RegisterBucketOrPanic(db, authValidatedBucket)
	return &storeImpl{
		DB: db,
	}
}
