// This file was originally generated with
// //go:generate cp ../../../../central/notifier/datastore/internal/store/bolt/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

var (
	notifierBucket = []byte("notifiers")
)

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, notifierBucket)
	return &storeImpl{
		DB: db,
	}
}

type storeImpl struct {
	*bolt.DB
}

// GetAll retrieves all notifiers from bolt
func (b *storeImpl) GetAll(_ context.Context) ([]*storage.Notifier, error) {
	var notifiers []*storage.Notifier
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(notifierBucket)
		return b.ForEach(func(k, v []byte) error {
			var notifier storage.Notifier
			if err := proto.Unmarshal(v, &notifier); err != nil {
				return err
			}
			notifiers = append(notifiers, &notifier)
			return nil
		})
	})
	return notifiers, err
}

func addUniqueCheck(tx *bolt.Tx, notifier *storage.Notifier) error {
	if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, notifierBucket, notifier.GetId(), notifier.GetName()); err != nil {
		return errors.Wrap(err, "Could not add notifier due to name validation")
	}
	return nil
}

func updateUniqueCheck(tx *bolt.Tx, notifier *storage.Notifier) error {
	if val, _ := secondarykey.GetCurrentUniqueKey(tx, notifierBucket, notifier.GetId()); val != notifier.GetName() {
		if err := secondarykey.UpdateUniqueKey(tx, notifierBucket, notifier.GetId(), notifier.GetName()); err != nil {
			return errors.Wrap(err, "Could not update auth provider due to name validation")
		}
	}
	return nil
}

// Upsert upserts a notifier to bolt
func (b *storeImpl) Upsert(_ context.Context, notifier *storage.Notifier) error {
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifierBucket)
		if bolthelper.Exists(bucket, notifier.GetId()) {
			// If it exists, then we are updating
			if err := updateUniqueCheck(tx, notifier); err != nil {
				return err
			}
		} else {
			if err := addUniqueCheck(tx, notifier); err != nil {
				return err
			}
		}

		bytes, err := proto.Marshal(notifier)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(notifier.GetId()), bytes)
	})
	return err
}
