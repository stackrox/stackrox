package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var namespaceBucket = []byte("namespaces")

// Store provides storage functionality for alerts.
type Store interface {
	GetNamespace(id string) (*storage.Namespace, bool, error)
	GetNamespaces() ([]*storage.Namespace, error)
	AddNamespace(*storage.Namespace) error
	UpdateNamespace(*storage.Namespace) error
	RemoveNamespace(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, namespaceBucket)
	return &storeImpl{
		DB: db,
	}
}
