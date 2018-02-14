package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const alertBucket = "alerts"

func (b *BoltDB) getAlert(id string, bucket *bolt.Bucket) (alert *v1.Alert, exists bool, err error) {
	alert = new(v1.Alert)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, alert)
	return
}

// GetAlert returns an alert with given id.
func (b *BoltDB) GetAlert(id string) (alert *v1.Alert, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		alert, exists, err = b.getAlert(id, bucket)
		return err
	})

	return
}

// GetAlerts ignores the request and gives all values
func (b *BoltDB) GetAlerts(*v1.GetAlertsRequest) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertBucket))
		return b.ForEach(func(k, v []byte) error {
			var alert v1.Alert
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			alerts = append(alerts, &alert)
			return nil
		})
	})
	return alerts, err
}

// AddAlert adds an alert into Bolt
func (b *BoltDB) AddAlert(alert *v1.Alert) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		_, exists, err := b.getAlert(alert.Id, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Alert %v cannot be added because it already exists", alert.GetId())
		}
		bytes, err := proto.Marshal(alert)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(alert.Id), bytes)
		return err
	})
}

// UpdateAlert upserts an alert into Bolt
func (b *BoltDB) UpdateAlert(alert *v1.Alert) error {
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

// RemoveAlert removes an alert into Bolt
func (b *BoltDB) RemoveAlert(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertBucket))
		err := b.Delete([]byte(id))
		return err
	})
}
