package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var (
	authProviderBucket = []byte("authProviders")
)

// Store stores and retrieves providers from the KV storage mechanism.
//go:generate mockgen-wrapper
type Store interface {
	GetAllAuthProviders() ([]*storage.AuthProvider, error)

	AddAuthProvider(authProvider *storage.AuthProvider) error
	UpdateAuthProvider(authProvider *storage.AuthProvider) error
	RemoveAuthProvider(d string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, authProviderBucket)
	return &storeImpl{
		db: db,
	}
}
