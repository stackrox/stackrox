package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const alertBucket = "alerts"

// GetAlert returns an alert with given id.
func (b *BoltDB) GetAlert(id string) (alert *v1.Alert, exists bool, err error) {
	alert = new(v1.Alert)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertBucket))
		val := b.Get([]byte(id))
		if val == nil {
			exists = false
			return nil
		}

		exists = true
		return proto.Unmarshal(val, alert)
	})

	return
}

// GetAlerts ignores the request and gives all values
func (b *BoltDB) GetAlerts(*v1.GetAlertsRequest) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertBucket))
		b.ForEach(func(k, v []byte) error {
			var alert v1.Alert
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			alerts = append(alerts, &alert)
			return nil
		})
		return nil
	})
	return alerts, err
}

func (b *BoltDB) upsertAlert(alert *v1.Alert) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertBucket))
		bytes, err := proto.Marshal(alert)
		if err != nil {
			return err
		}
		err = b.Put([]byte(alert.Id), bytes)
		return err
	})
}

// AddAlert upserts an alert into Bolt
func (b *BoltDB) AddAlert(alert *v1.Alert) error {
	return b.upsertAlert(alert)
}

// UpdateAlert upserts an alert into Bolt
func (b *BoltDB) UpdateAlert(alert *v1.Alert) error {
	return b.upsertAlert(alert)
}

// RemoveAlert removes an alert into Bolt
func (b *BoltDB) RemoveAlert(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertBucket))
		err := b.Delete([]byte(id))
		return err
	})
}
