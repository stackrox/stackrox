package bolt

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/notifier/datastore/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/dberrors"
	ops "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

var (
	notifierBucket = []byte("notifiers")
)

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) store.Store {
	bolthelper.RegisterBucketOrPanic(db, notifierBucket)
	return &storeImpl{
		DB: db,
	}
}

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getNotifier(id string, bucket *bolt.Bucket) (notifier *storage.Notifier, exists bool, err error) {
	notifier = new(storage.Notifier)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, notifier)
	return
}

// Get returns notifier with given id.
func (b *storeImpl) Get(_ context.Context, id string) (notifier *storage.Notifier, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Notifier")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifierBucket)
		notifier, exists, err = b.getNotifier(id, bucket)
		return err
	})
	return
}

// GetAll retrieves all notifiers from bolt
func (b *storeImpl) GetAll(_ context.Context) ([]*storage.Notifier, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Notifier")
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

func (b *storeImpl) Exists(_ context.Context, id string) (bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Exists, "Notifier")

	var exists bool
	err := b.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(notifierBucket).Get([]byte(id)) != nil
		return nil
	})
	return exists, err
}

// Upsert upserts a notifier to bolt
func (b *storeImpl) Upsert(_ context.Context, notifier *storage.Notifier) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Notifier")

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

// Delete removes a notifier.
func (b *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Notifier")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(notifierBucket)
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Notifier", ID: string(key)}
		}
		if err := secondarykey.RemoveUniqueKey(tx, notifierBucket, id); err != nil {
			return err
		}
		return b.Delete(key)
	})
}
