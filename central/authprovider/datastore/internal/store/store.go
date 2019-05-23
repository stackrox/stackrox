package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	authProviderBucket = []byte("authProviders")
)

// Store stores and retrieves providers from the KV storage mechanism.
//go:generate mockgen-wrapper Store
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
