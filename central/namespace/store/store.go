package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
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
