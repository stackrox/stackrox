package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const namespaceBucket = "namespaces"

// Store provides storage functionality for alerts.
type Store interface {
	GetNamespace(id string) (*v1.Namespace, bool, error)
	GetNamespaces() ([]*v1.Namespace, error)
	AddNamespace(*v1.Namespace) error
	UpdateNamespace(*v1.Namespace) error
	RemoveNamespace(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, namespaceBucket)
	return &storeImpl{
		DB: db,
	}
}
