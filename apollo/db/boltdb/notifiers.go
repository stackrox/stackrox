package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const notifierBucket = "notifiers"

// GetNotifier returns notifier with given id.
func (b *BoltDB) GetNotifier(name string) (notifier *v1.Notifier, exists bool, err error) {
	notifier = new(v1.Notifier)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(notifierBucket))
		val := b.Get([]byte(name))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, notifier)
	})

	return
}

// GetNotifiers retrieves notifiers matching the request from bolt
func (b *BoltDB) GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error) {
	var notifiers []*v1.Notifier
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(notifierBucket))
		b.ForEach(func(k, v []byte) error {
			var notifier v1.Notifier
			if err := proto.Unmarshal(v, &notifier); err != nil {
				return err
			}
			notifiers = append(notifiers, &notifier)
			return nil
		})
		return nil
	})
	return notifiers, err
}

func (b *BoltDB) upsertNotifier(notifier *v1.Notifier) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(notifierBucket))
		bytes, err := proto.Marshal(notifier)
		if err != nil {
			return err
		}
		err = b.Put([]byte(notifier.Name), bytes)
		return err
	})
}

// AddNotifier adds a notifier to bolt
func (b *BoltDB) AddNotifier(notifier *v1.Notifier) error {
	return b.upsertNotifier(notifier)
}

// UpdateNotifier updates a notifier to bolt
func (b *BoltDB) UpdateNotifier(notifier *v1.Notifier) error {
	return b.upsertNotifier(notifier)
}

// RemoveNotifier removes a notifier.
func (b *BoltDB) RemoveNotifier(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(notifierBucket))
		return b.Delete([]byte(name))
	})
}
