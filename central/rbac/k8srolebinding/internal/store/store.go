package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	// roleBindingBucket is the bucket tht stores role bindings objects.
	roleBindingBucket = []byte("rolebindings")
)

// Store provides access and update functions for role bindings.
//go:generate mockgen-wrapper Store
type Store interface {
	ListAllRoleBindings() ([]*storage.K8SRoleBinding, error)

	GetRoleBinding(id string) (*storage.K8SRoleBinding, bool, error)
	UpsertRoleBinding(rolebinding *storage.K8SRoleBinding) error
	RemoveRoleBinding(id string) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, roleBindingBucket)
	return &storeImpl{
		db: db,
	}
}
