package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const groupsBucket = "groups"

// Store updates and utilizes groups, which are attribute to role mappings.
type Store interface {
	Get(props *v1.GroupProperties) (*v1.Group, error)
	GetAll() ([]*v1.Group, error)

	Walk(authProviderID string, attributes map[string][]string) ([]*v1.Group, error)

	Add(*v1.Group) error
	Update(*v1.Group) error
	Upsert(*v1.Group) error
	Remove(props *v1.GroupProperties) error
}

// New returns a new instance of a Store.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, groupsBucket)

	return &storeImpl{
		db: db,
	}
}
