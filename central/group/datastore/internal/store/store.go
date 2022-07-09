package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

var groupsBucket = []byte("groups2")

var isEmptyGroupPropertiesF = func(props *storage.GroupProperties) bool {
	if props.GetAuthProviderId() == "" && props.GetKey() == "" && props.GetValue() == "" {
		return true
	}
	return false
}

// Store updates and utilizes groups, which are attribute to role mappings.
//go:generate mockgen-wrapper
type Store interface {
	Get(props *storage.GroupProperties) (*storage.Group, error)
	GetFiltered(func(*storage.GroupProperties) bool) ([]*storage.Group, error)
	GetAll() ([]*storage.Group, error)

	Walk(authProviderID string, attributes map[string][]string) ([]*storage.Group, error)

	Add(*storage.Group) error
	Update(*storage.Group) error
	Mutate(remove, update, add []*storage.Group) error
	Remove(props *storage.GroupProperties) error
}

// New returns a new instance of a Store.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, groupsBucket)

	store := &storeImpl{
		db: db,
	}
	grps, err := store.GetFiltered(isEmptyGroupPropertiesF)
	utils.Should(err)
	for _, grp := range grps {
		err = store.Remove(grp.GetProps())
		utils.Should(err)
	}

	return store
}
