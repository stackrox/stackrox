// This file was originally generated with
// //go:generate cp central/group/datastore/internal/store/bolt/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var groupsBucket = []byte("groups2")

// New returns a new instance of a Store.
func New(db *bolt.DB) Store {
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

func upsertInTransaction(tx *bolt.Tx, groups ...*storage.Group) error {
	bucket := tx.Bucket(groupsBucket)

	for _, group := range groups {
		bytes, err := proto.Marshal(group)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal group %v", group)
		}

		if err := bucket.Put([]byte(group.GetProps().GetId()), bytes); err != nil {
			return err
		}
	}
	return nil
}
