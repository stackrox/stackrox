package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var groupsBucket = []byte("groups")

// Store updates and utilizes groups, which are attribute to role mappings.
//go:generate mockgen-wrapper Store
type Store interface {
	Get(props *storage.GroupProperties) (*storage.Group, error)
	GetAll() ([]*storage.Group, error)

	Walk(authProviderID string, attributes map[string][]string) ([]*storage.Group, error)

	Add(*storage.Group) error
	Update(*storage.Group) error
	Upsert(*storage.Group) error
	Mutate(remove, update, add []*storage.Group) error
	Remove(props *storage.GroupProperties) error
}

// New returns a new instance of a Store.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, groupsBucket)

	return &storeImpl{
		db: db,
	}
}
