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
func (s *storeImpl) Get(props *storage.GroupProperties) (grp *storage.Group, err error) {
	if props.GetId() == "" {
		return s.getByProps(props)
	}
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		v := buc.Get([]byte(props.GetId()))
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

// getByProps returns a group matching the given properties if it exists from the store.
// If more than one group is found matching the properties, an error will be returned.
// TODO: This can be removed once retrieving the group by its properties is fully deprecated.
func (s *storeImpl) getByProps(props *storage.GroupProperties) (grp *storage.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		var groups []*storage.Group
		groups, err = filterInTransaction(tx, func(p *storage.GroupProperties) bool {
			if props.GetAuthProviderId() != p.GetAuthProviderId() {
				return false
			}
			if props.GetKey() != p.GetKey() {
				return false
			}
			if props.GetValue() != p.GetValue() {
				return false
			}
			return true
		})

		if len(groups) > 1 {
			return errox.InvalidArgs.Newf("multiple groups found for properties (auth provider id=%s, key=%s, "+
				"value=%s), provide an ID to retrieve a group unambiguously",
				props.GetAuthProviderId(), props.GetKey(), props.GetValue())
		}
		// If no groups are found, return nil, mimicking the behavior of Get().
		if len(groups) == 0 {
			return nil
		}
		grp = groups[0]
		return nil
	})
	return grp, err
}

func (s *storeImpl) GetFiltered(filter func(*storage.GroupProperties) bool) (groups []*storage.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		groups, err = filterInTransaction(tx, filter)
		return err
	})
	return groups, err
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
			grpss, err := filterInTransaction(tx, func(props *storage.GroupProperties) bool {
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

// Remove removes the group with matching properties from the store.
// Returns an error if no such group exists.
func (s *storeImpl) Remove(props *storage.GroupProperties) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return removeInTransaction(tx, props)
	})
}

// Mutate does a set of mutations to the store in a single transaction, returning an error if any affected
// state is unexpected.
func (s *storeImpl) Mutate(toRemove, toUpdate, toAdd []*storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, group := range toRemove {
			if err := removeInTransaction(tx, group.GetProps()); err != nil {
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

	// TODO: Once the deprecation of retrieving groups by their properties is fully deprecated, this condition
	// can be removed and groups shall only be retrievable via their id.
	if id != "" {
		if buc.Get([]byte(id)) == nil {
			return errox.NotFound.Newf("group config for %q does not exist", id)
		}
	} else {
		grps, err := filterInTransaction(tx, func(props *storage.GroupProperties) bool {
			if group.GetProps().GetAuthProviderId() != props.GetAuthProviderId() {
				return false
			}
			if group.GetProps().GetKey() != props.GetKey() {
				return false
			}
			if group.GetProps().GetValue() != props.GetValue() {
				return false
			}
			return true
		})
		if err != nil {
			return err
		}
		if len(grps) > 1 {
			return errox.InvalidArgs.Newf("multiple groups found for properties (auth provider id=%s, key=%s, "+
				"value=%s), provide an ID to retrieve a group unambiguously",
				group.GetProps().GetAuthProviderId(), group.GetProps().GetKey(), group.GetProps().GetValue())
		}
		if len(grps) == 0 {
			return errox.NotFound.Newf("group config for (auth provider id=%s, key=%s, value=%s) does not exist",
				group.GetProps().GetAuthProviderId(), group.GetProps().GetKey(), group.GetProps().GetValue())
		}
		id = grps[0].GetProps().GetId()
	}

	bytes, err := proto.Marshal(group)
	if err != nil {
		return errox.InvariantViolation.CausedBy(err)
	}

	return buc.Put([]byte(id), bytes)
}

func removeInTransaction(tx *bolt.Tx, props *storage.GroupProperties) error {
	buc := tx.Bucket(groupsBucket)
	id := props.GetId()

	// TODO: Once the deprecation of retrieving groups by their properties is fully deprecated, this condition
	// can be removed and groups shall only be retrievable via their id.
	if id != "" {
		if buc.Get([]byte(id)) == nil {
			return errox.NotFound.Newf("group config for %q does not exist", id)
		}
	} else {
		grps, err := filterInTransaction(tx, func(p *storage.GroupProperties) bool {
			if props.GetAuthProviderId() != p.GetAuthProviderId() {
				return false
			}
			if props.GetKey() != p.GetKey() {
				return false
			}
			if props.GetValue() != p.GetValue() {
				return false
			}
			return true
		})
		if err != nil {
			return err
		}
		if len(grps) > 1 {
			return errox.InvalidArgs.Newf("multiple groups found for properties (auth provider id=%s, key=%s, "+
				"value=%s), provide an ID to retrieve a group unambiguously",
				props.GetAuthProviderId(), props.GetKey(), props.GetValue())
		}
		if len(grps) == 0 {
			return errox.NotFound.Newf("group config for (auth provider id=%s, key=%s, value=%s) does not exist",
				props.GetAuthProviderId(), props.GetKey(), props.GetValue())
		}
		id = grps[0].GetProps().GetId()
	}

	return buc.Delete([]byte(id))
}

func filterInTransaction(tx *bolt.Tx, filter func(*storage.GroupProperties) bool) (grps []*storage.Group, err error) {
	buc := tx.Bucket(groupsBucket)

	err = buc.ForEach(func(k, v []byte) error {
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
	return grps, err
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
