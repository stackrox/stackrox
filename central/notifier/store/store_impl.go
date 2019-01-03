package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/uuid"
)

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

// GetNotifier returns notifier with given id.
func (b *storeImpl) GetNotifier(id string) (notifier *storage.Notifier, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Notifier")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifierBucket)
		notifier, exists, err = b.getNotifier(id, bucket)
		return err
	})
	return
}

// GetNotifiers retrieves notifiers matching the request from bolt
func (b *storeImpl) GetNotifiers(request *v1.GetNotifiersRequest) ([]*storage.Notifier, error) {
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

// AddNotifier adds a notifier to bolt
func (b *storeImpl) AddNotifier(notifier *storage.Notifier) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Notifier")
	notifier.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifierBucket)
		_, exists, err := b.getNotifier(notifier.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Notifier %v (%v) cannot be added because it already exists", notifier.GetName(), notifier.GetId())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, notifierBucket, notifier.GetId(), notifier.GetName()); err != nil {
			return fmt.Errorf("Could not add notifier due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(notifier)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(notifier.GetId()), bytes)
	})
	return notifier.Id, err
}

// UpdateNotifier updates a notifier to bolt
func (b *storeImpl) UpdateNotifier(notifier *storage.Notifier) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Notifier")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(notifierBucket)
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, notifierBucket, notifier.GetId()); val != notifier.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, notifierBucket, notifier.GetId(), notifier.GetName()); err != nil {
				return fmt.Errorf("Could not update notifier due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(notifier)
		if err != nil {
			return err
		}
		return b.Put([]byte(notifier.GetId()), bytes)
	})
}

// RemoveNotifier removes a notifier.
func (b *storeImpl) RemoveNotifier(id string) error {
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
