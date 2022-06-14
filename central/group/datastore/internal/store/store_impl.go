package store

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	bolt "go.etcd.io/bbolt"
)

// We use custom serialization for speed since this store will need to be 'Walked'
// to find all of the roles that apply to a given user.
type storeImpl struct {
	db *bolt.DB
}

// Get returns a group matching the given properties if it exists from the store.
func (s *storeImpl) Get(id string) (grp *storage.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		v := buc.Get([]byte(id))
		if v == nil {
			return nil
		}
		var err error
		var marshalledGroup storage.Group
		err = proto.Unmarshal(v, &marshalledGroup)
		grp = &marshalledGroup
		return err
	})
	return
}

func (s *storeImpl) GetFiltered(filter func(*storage.GroupProperties) bool) ([]*storage.Group, error) {
	var grps []*storage.Group
	err := s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		return buc.ForEach(func(k, v []byte) error {
			var grp storage.Group
			err := proto.Unmarshal(v, &grp)
			if err != nil {
				return err
			}
			if filter == nil || filter(grp.GetProps()) {
				grps = append(grps, &grp)
			}
			return nil
		})
	})
	return grps, err
}

// GetAll return all groups currently in the store.
func (s *storeImpl) GetAll() (grps []*storage.Group, err error) {
	return s.GetFiltered(nil)
}

// Walk is an optimization that allows to search through the datastore and find
// groups that apply to a user within a single transaction.
func (s *storeImpl) Walk(authProviderID string, attributes map[string][]string) (grps []*storage.Group, err error) {
	// Which groups to search for based on the auth provider and attributes.
	toSearch := getPossibleGroupProperties(authProviderID, attributes)

	// Search for groups in the list.
	err = s.db.View(func(tx *bolt.Tx) error {
		for _, check := range toSearch {
			grpss, err := s.GetFiltered(func(props *storage.GroupProperties) bool {
				if check.GetAuthProviderId() != props.GetAuthProviderId() {
					return false
				}
				if check.GetKey() != props.GetKey() {
					return false
				}
				if check.GetValue() != props.GetValue() {
					return false
				}
				return true
			})
			if err != nil {
				return err
			}
			grps = append(grps, grpss...)
		}
		return nil
	})
	return
}

// Add adds a group to the store.
// Returns an error if a group with the same properties already exists.
func (s *storeImpl) Add(group *storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return addInTransaction(tx, group)
	})
}

// Update updates a group in the store.
// Returns an error if a group with the same properties does not already exist.
func (s *storeImpl) Update(group *storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return updateInTransaction(tx, group)
	})
}

// Upsert adds or updates a group in the store.
func (s *storeImpl) Upsert(group *storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		id := group.GetProps().GetId()
		bytes, err := proto.Marshal(group)
		if err != nil {
			return errox.InvariantViolation.CausedBy(err)
		}
		return buc.Put([]byte(id), bytes)
	})
}

// Remove removes the group with matching properties from the store.
// Returns an error if no such group exists.
func (s *storeImpl) Remove(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return removeInTransaction(tx, id)
	})
}

// Mutate does a set of mutations to the store in a single transaction, returning an error if any affected
// state is unexpected.
func (s *storeImpl) Mutate(toRemove, toUpdate, toAdd []*storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, group := range toRemove {
			if err := removeInTransaction(tx, group.GetProps().GetId()); err != nil {
				return errors.Wrap(err, "error removing during mutation")
			}
		}
		for _, group := range toUpdate {
			if err := updateInTransaction(tx, group); err != nil {
				return errors.Wrap(err, "error updating during mutation")
			}
		}
		for _, group := range toAdd {
			if err := addInTransaction(tx, group); err != nil {
				return errors.Wrap(err, "error adding during mutation")
			}
		}
		return nil
	})
}

// Helpers
//////////

func addInTransaction(tx *bolt.Tx, group *storage.Group) error {
	id := group.GetProps().GetId()

	buc := tx.Bucket(groupsBucket)
	if buc.Get([]byte(id)) != nil {
		return errox.AlreadyExists.Newf("group config for %q already exists", id)
	}

	bytes, err := proto.Marshal(group)
	if err != nil {
		return errox.InvariantViolation.CausedBy(err)
	}

	return buc.Put([]byte(id), bytes)
}

func updateInTransaction(tx *bolt.Tx, group *storage.Group) error {
	id := group.GetProps().GetId()

	buc := tx.Bucket(groupsBucket)
	if buc.Get([]byte(id)) == nil {
		return errox.NotFound.Newf("group config for %q does not exist", id)
	}

	bytes, err := proto.Marshal(group)
	if err != nil {
		return errox.InvariantViolation.CausedBy(err)
	}

	return buc.Put([]byte(id), bytes)
}

func removeInTransaction(tx *bolt.Tx, id string) error {
	buc := tx.Bucket(groupsBucket)
	if buc.Get([]byte(id)) == nil {
		return errox.NotFound.Newf("group config for %q does not exist", id)
	}
	return buc.Delete([]byte(id))
}

// When given an auth provider and attributes, we will look for all keys and
// key/value pairs that exist in the datastore for the given auth provider.
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
