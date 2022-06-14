package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var groupsBucket = []byte("groups2")

// Store updates and utilizes groups, which are attribute to role mappings.
//go:generate mockgen-wrapper
type Store interface {
	Get(props *storage.GroupProperties) (*storage.Group, error)
	GetFiltered(func(*storage.GroupProperties) bool) ([]*storage.Group, error)
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

	store := &storeImpl{
		db: db,
	}

	allEmptyGroupProperty := storage.GroupProperties{AuthProviderId: "", Key: "", Value: ""}
	_ = store.Remove(&allEmptyGroupProperty) // ignore error to suppress warning

	return store
}
