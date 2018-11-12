package store

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
)

// We use custom serialization for speed since this store will need to be 'Walked'
// to find all of the roles that apply to a given user.
type storeImpl struct {
	db *bolt.DB
}

// Get returns a group matching the given properties if it exists from the store.
func (s *storeImpl) Get(props *v1.GroupProperties) (grp *v1.Group, err error) {
	s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
		k := serializeKey(props)
		v := buc.Get(k)
		if v == nil {
			return nil
		}
		grp, err = deserialize(k, v)
		return nil
	})
	return
}

// GetAll return all groups currently in the store.
func (s *storeImpl) GetAll() (grps []*v1.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
		buc.ForEach(func(k, v []byte) error {
			grp, err := deserialize(k, v)
			if err != nil {
				return err
			}
			grps = append(grps, grp)
			return nil
		})
		return nil
	})
	return
}

// Walk is an optimization. Since we normally want to find groups that apply to a user,
// we need to search through the DB and find all of them. Here we do the search with a single
// transaction.
//
// When given an auth provider, and map, we will look for all key and key/value pairs that exist
// in the store both for the given auth provider, and for no auth provider (applies to all auth providers.)
func (s *storeImpl) Walk(authProviderID string, attributes map[string][]string) (grps []*v1.Group, err error) {
	// Build list to search
	toSearch := getPossibleGroupProperties(authProviderID, attributes)

	// Search for items in list.
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
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
func (s *storeImpl) Add(group *v1.Group) error {
	key, value := serialize(group)

	return s.db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
		if buc.Get(key) != nil {
			return fmt.Errorf("group config for %s already exists", string(key))
		}
		buc.Put(key, value)
		return nil
	})
}

// Update updates a group in the store.
// Returns an error if a group with the same properties does not already exist.
func (s *storeImpl) Update(group *v1.Group) error {
	key, value := serialize(group)

	return s.db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
		if buc.Get(key) == nil {
			return fmt.Errorf("group config for %s does not exist", string(key))
		}
		buc.Put(key, value)
		return nil
	})
}

// Upsert adds or updates a group in the store.
func (s *storeImpl) Upsert(group *v1.Group) error {
	key, value := serialize(group)

	return s.db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
		buc.Put(key, value)
		return nil
	})
}

// Remove removes the group with matching properties from the store.
// Does not return an error if no such group exists.
func (s *storeImpl) Remove(props *v1.GroupProperties) error {
	key := serializeKey(props)

	return s.db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket([]byte(groupsBucket))
		buc.Delete(key)
		return nil
	})
}

// Helpers
//////////

func getPossibleGroupProperties(authProviderID string, attributes map[string][]string) (props []*v1.GroupProperties) {
	// We need to consider no provider, and the provider given.
	possibleAuthProviders := []string{"", authProviderID}
	for _, ap := range possibleAuthProviders {
		// Need to consider no key.
		props = append(props, &v1.GroupProperties{AuthProviderId: ap})
		for key, values := range attributes {
			// Need to consider key with no value
			props = append(props, &v1.GroupProperties{AuthProviderId: ap, Key: key})
			// Consider all Key/Value pairs present.
			for _, value := range values {
				props = append(props, &v1.GroupProperties{AuthProviderId: ap, Key: key, Value: value})
			}
		}
	}
	return
}
