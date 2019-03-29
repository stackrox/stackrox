package store

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// We use custom serialization for speed since this store will need to be 'Walked'
// to find all of the roles that apply to a given user.
type storeImpl struct {
	db *bolt.DB
}

// Get returns a group matching the given properties if it exists from the store.
func (s *storeImpl) Get(props *storage.GroupProperties) (grp *storage.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		k := serializeKey(props)
		v := buc.Get(k)
		if v == nil {
			return nil
		}
		var err error
		grp, err = deserialize(k, v)
		return err
	})
	return
}

// GetAll return all groups currently in the store.
func (s *storeImpl) GetAll() (grps []*storage.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		return buc.ForEach(func(k, v []byte) error {
			grp, err := deserialize(k, v)
			if err != nil {
				return err
			}
			grps = append(grps, grp)
			return nil
		})
	})
	return
}

// Walk is an optimization. Since we normally want to find groups that apply to a user,
// we need to search through the DB and find all of them. Here we do the search with a single
// transaction.
//
// When given an auth provider, and map, we will look for all key and key/value pairs that exist
// in the store both for the given auth provider, and for no auth provider (applies to all auth providers.)
func (s *storeImpl) Walk(authProviderID string, attributes map[string][]string) (grps []*storage.Group, err error) {
	// Build list to search
	toSearch := getPossibleGroupProperties(authProviderID, attributes)

	// Search for items in list.
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		for _, check := range toSearch {
			serializedKey := serializeKey(check)
			if serializedVal := buc.Get(serializedKey); serializedVal != nil {
				grp, err := deserialize(serializedKey, serializedVal)
				if err != nil {
					return err
				}
				grps = append(grps, grp)
			}
		}
		return nil
	})
	return
}

// Add adds a group to the store.
// Returns an error if a group with the same properties already exists.
func (s *storeImpl) Add(group *storage.Group) error {
	key, value := serialize(group)

	return s.db.Update(func(tx *bolt.Tx) error {
		return addInTransaction(tx, key, value)
	})
}

// Update updates a group in the store.
// Returns an error if a group with the same properties does not already exist.
func (s *storeImpl) Update(group *storage.Group) error {
	key, value := serialize(group)

	return s.db.Update(func(tx *bolt.Tx) error {
		return updateInTransaction(tx, key, value)
	})
}

// Upsert adds or updates a group in the store.
func (s *storeImpl) Upsert(group *storage.Group) error {
	key, value := serialize(group)

	return s.db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		return buc.Put(key, value)
	})
}

// Remove removes the group with matching properties from the store.
// Returns an error if no such group exists.
func (s *storeImpl) Remove(props *storage.GroupProperties) error {
	key := serializeKey(props)

	return s.db.Update(func(tx *bolt.Tx) error {
		return removeInTransaction(tx, key)
	})
}

// Mutate does a set of mutations to the store in a single transaction, returning an error if any affected
// state is unexpected.
func (s *storeImpl) Mutate(toRemove, toUpdate, toAdd []*storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, group := range toRemove {
			key := serializeKey(group.GetProps())
			if err := removeInTransaction(tx, key); err != nil {
				return errors.Wrap(err, "error removing during mutation")
			}
		}
		for _, group := range toUpdate {
			key, value := serialize(group)
			if err := updateInTransaction(tx, key, value); err != nil {
				return errors.Wrap(err, "error updating during mutation")
			}
		}
		for _, group := range toAdd {
			key, value := serialize(group)
			if err := addInTransaction(tx, key, value); err != nil {
				return errors.Wrap(err, "error adding during mutation")
			}
		}
		return nil
	})
}

// Helpers
//////////

func addInTransaction(tx *bolt.Tx, key, value []byte) error {
	buc := tx.Bucket(groupsBucket)
	if buc.Get(key) != nil {
		return fmt.Errorf("group config for %s already exists", key)
	}
	return buc.Put(key, value)
}

func updateInTransaction(tx *bolt.Tx, key, value []byte) error {
	buc := tx.Bucket(groupsBucket)
	if buc.Get(key) == nil {
		return fmt.Errorf("group config for %s does not exist", key)
	}
	return buc.Put(key, value)
}

func removeInTransaction(tx *bolt.Tx, key []byte) error {
	buc := tx.Bucket(groupsBucket)
	if buc.Get(key) == nil {
		return fmt.Errorf("group config for %s does not exist", key)
	}
	return buc.Delete(key)
}

func getPossibleGroupProperties(authProviderID string, attributes map[string][]string) (props []*storage.GroupProperties) {
	// Need to consider no key.
	props = append(props, &storage.GroupProperties{AuthProviderId: authProviderID})
	for key, values := range attributes {
		// Need to consider key with no value
		props = append(props, &storage.GroupProperties{AuthProviderId: authProviderID, Key: key})
		// Consider all Key/Value pairs present.
		for _, value := range values {
			props = append(props, &storage.GroupProperties{AuthProviderId: authProviderID, Key: key, Value: value})
		}
	}
	return
}
