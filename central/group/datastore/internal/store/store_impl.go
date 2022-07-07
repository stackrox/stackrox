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
// TODO(ROX-11592): This can be removed once retrieving the group by its properties is fully deprecated.
func (s *storeImpl) getByProps(props *storage.GroupProperties) (grp *storage.Group, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		grp, err := getByPropsInTransaction(tx, props)
		if err != nil {
			return err
		}
		// If no groups are found, return nil, mimicking the behavior of Get().
		if grp == nil {
			return nil
		}
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
			grpss, err := filterByPropsInTransaction(tx, check)
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

	// Check whether the to-be-added group is a default group, ensure that it does not yet exist.
	defaultGroupExists, err := checkDefaultGroupForProps(tx, group.GetProps())
	if err != nil {
		return err
	}

	if defaultGroupExists {
		return errox.AlreadyExists.Newf("a default group already exists for auth provider %q",
			group.GetProps().GetAuthProviderId())
	}

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

	// Check whether the to-be-added group is a default group, ensure that it does not yet exist.
	defaultGroupExists, err := checkDefaultGroupForProps(tx, group.GetProps())
	if err != nil {
		return err
	}

	if defaultGroupExists {
		return errox.AlreadyExists.Newf("a default group already exists for auth provider %q",
			group.GetProps().GetAuthProviderId())
	}

	// TODO(ROX-11592): Once the deprecation of retrieving groups by their properties is fully deprecated, this condition
	// can be removed and groups shall only be retrievable via their id.
	if id != "" {
		if buc.Get([]byte(id)) == nil {
			return errox.NotFound.Newf("group config for %q does not exist", id)
		}
	} else {
		grp, err := getByPropsInTransaction(tx, group.GetProps())
		if err != nil {
			return err
		}
		if grp == nil {
			return errox.NotFound.Newf("group config for (auth provider id=%s, key=%s, value=%s) does not exist",
				group.GetProps().GetAuthProviderId(), group.GetProps().GetKey(), group.GetProps().GetValue())
		}
		id = grp.GetProps().GetId()
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

	// TODO(ROX-11592): Once the deprecation of retrieving groups by their properties is fully deprecated, this condition
	// can be removed and groups shall only be retrievable via their id.
	if id != "" {
		if buc.Get([]byte(id)) == nil {
			return errox.NotFound.Newf("group config for %q does not exist", id)
		}
	} else {
		grp, err := getByPropsInTransaction(tx, props)
		if err != nil {
			return err
		}
		if grp == nil {
			return errox.NotFound.Newf("group config for (auth provider id=%s, key=%s, value=%s) does not exist",
				props.GetAuthProviderId(), props.GetKey(), props.GetValue())
		}
		id = grp.GetProps().GetId()
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

func filterByPropsInTransaction(tx *bolt.Tx, props *storage.GroupProperties) ([]*storage.Group, error) {
	grps, err := filterInTransaction(tx, func(stored *storage.GroupProperties) bool {
		if props.GetAuthProviderId() != stored.GetAuthProviderId() ||
			props.GetKey() != stored.GetKey() ||
			props.GetValue() != stored.GetValue() {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return grps, nil
}

func getByPropsInTransaction(tx *bolt.Tx, props *storage.GroupProperties) (*storage.Group, error) {
	grps, err := filterByPropsInTransaction(tx, props)
	if err != nil {
		return nil, err
	}
	if len(grps) == 0 {
		return nil, nil
	}

	if len(grps) > 1 {
		return nil, errox.InvalidArgs.Newf("multiple groups found for properties (auth provider id=%s, key=%s, "+
			"value=%s), provide an ID to retrieve a group unambiguously",
			props.GetAuthProviderId(), props.GetKey(), props.GetValue())
	}

	return grps[0], nil
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

// checkDefaultGroupForProps will check whether the given properties are a default group and, if they are, search the
// store for the given auth provider ID, checking whether a default group already exists.
// If the properties do not indicate a default group or the default group does not yet exist, it will return false.
// Otherwise, it will return true.
func checkDefaultGroupForProps(tx *bolt.Tx, props *storage.GroupProperties) (bool, error) {
	// 1. Short-circuit if the props do not indicate a default group. A default group only has the auth provider ID
	// field set.
	if !isDefaultGroup(props) {
		return false, nil
	}

	// 2. Filter for the default group.
	grp, err := getByPropsInTransaction(tx, &storage.GroupProperties{AuthProviderId: props.GetAuthProviderId()})
	if err != nil {
		return false, err
	}
	defaultGroupExists := grp != nil
	return defaultGroupExists, nil
}

// isDefaultGroup will check whether the given properties are a default group.
// A default group won't have the key and value fields set, only the auth provider ID field.
func isDefaultGroup(props *storage.GroupProperties) bool {
	return props.GetKey() == "" && props.GetValue() == ""
}
