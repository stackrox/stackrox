package bolt

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errox"
	bolt "go.etcd.io/bbolt"
)

var groupsBucket = []byte("groups2")

// New returns a new instance of a Store.
func New(db *bolt.DB) store.Store {
	bolthelper.RegisterBucketOrPanic(db, groupsBucket)
	return &storeImpl{
		db: db,
	}
}

// We use custom serialization for speed since this store will need to be 'Walked'
// to find all the roles that apply to a given user.
type storeImpl struct {
	db *bolt.DB
}

func (s *storeImpl) getGroup(id string, bucket *bolt.Bucket) (group *storage.Group, exists bool, err error) {
	v := bucket.Get([]byte(id))
	if v == nil {
		return
	}
	exists = true
	group = new(storage.Group)
	err = proto.Unmarshal(v, group)
	return
}

// Get returns a group matching the given properties if it exists from the store.
func (s *storeImpl) Get(_ context.Context, propsID string) (group *storage.Group, exists bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		buc := tx.Bucket(groupsBucket)
		group, exists, err = s.getGroup(propsID, buc)
		return err
	})
	return
}

// GetAll return all groups currently in the store.
func (s *storeImpl) GetAll(ctx context.Context) (groups []*storage.Group, err error) {
	err = s.Walk(ctx, func(g *storage.Group) error {
		groups = append(groups, g)
		return nil
	})
	return groups, err
}

// Walk iterates over all the objects in the store and applies the closure
func (s *storeImpl) Walk(_ context.Context, fn func(obj *storage.Group) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(groupsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var group storage.Group
			if err := proto.Unmarshal(v, &group); err != nil {
				return err
			}
			return fn(&group)
		})
	})
}

// Upsert upserts a group to the store
func (s *storeImpl) Upsert(_ context.Context, group *storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return upsertInTransaction(tx, group)
	})
}

// UpsertMany upserts multiple groups to the store
func (s *storeImpl) UpsertMany(_ context.Context, groups []*storage.Group) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return upsertInTransaction(tx, groups...)
	})
}

// Delete removes the group with the specified propsID.
// Returns an error if no such group exists.
func (s *storeImpl) Delete(_ context.Context, propsID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return deleteInTransaction(tx, propsID)
	})
}

// DeleteMany removes multiple groups from the store given their ids.
func (s *storeImpl) DeleteMany(_ context.Context, ids []string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return deleteInTransaction(tx, ids...)
	})
}

// Helpers
//////////

func upsertInTransaction(tx *bolt.Tx, groups ...*storage.Group) error {
	bucket := tx.Bucket(groupsBucket)

	for _, group := range groups {
		bytes, err := proto.Marshal(group)
		if err != nil {
			return errox.InvariantViolation.CausedBy(err)
		}

		if err := bucket.Put([]byte(group.GetProps().GetId()), bytes); err != nil {
			return err
		}
	}
	return nil
}

func deleteInTransaction(tx *bolt.Tx, ids ...string) error {
	bucket := tx.Bucket(groupsBucket)

	for _, propsID := range ids {
		key := []byte(propsID)
		if bucket.Get(key) == nil {
			return errox.NotFound.Newf("group config for %q does not exist", propsID)
		}
		if err := bucket.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
