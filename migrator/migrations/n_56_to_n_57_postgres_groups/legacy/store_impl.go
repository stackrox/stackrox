// This file was originally generated with
// //go:generate cp central/group/datastore/internal/store/bolt/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var (
	groupsBucket = []byte("groups2")
	log          = loghelper.LogWrapper{}
)

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
			// Due to issues within the 105_106 migration, it's currently undefined which format you receive: either
			// the old format, which uses a composite key of auth provider, key, value as the key or the new format
			// which uses a UUID as key. Reasoning is that the old values were not cleaned up correctly. Skip values
			// in case the key can be deserialized to the old format.
			if props, err := deserializePropsKey(k); err == nil {
				log.WriteToStderrf("Found group with old format (%s), skipping this entry", props.String())
				return nil
			}
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

// UpsertOldFormat upserts a group to the store using the old format, i.e. using a composite key as unique identifier.
// This is only used for testing purposes.
func (s *storeImpl) UpsertOldFormat(_ context.Context, group *storage.Group) error {
	k, v := serialize(group)
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(groupsBucket)
		return bucket.Put(k, v)
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
