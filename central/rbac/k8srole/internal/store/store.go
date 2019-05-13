package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	// roleBucket is the bucket that stores k8s role objects.
	roleBucket = []byte("k8sroles")
)

// Store provides access and update functions for k8s roles.
//go:generate mockgen-wrapper Store
type Store interface {
	ListRoles(id []string) ([]*storage.K8SRole, error)
	ListAllRoles() ([]*storage.K8SRole, error)

	CountRoles() (int, error)
	GetRole(id string) (*storage.K8SRole, bool, error)
	UpsertRole(secret *storage.K8SRole) error
	RemoveRole(id string) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, roleBucket)
	return &storeImpl{
		db: db,
	}
}
