package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

// GetAlert returns an alert with given id.
func (b *storeImpl) ListAlert(id string) (alert *v1.ListAlert, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ListAlert")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertListBucket))
		alert = new(v1.ListAlert)
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		err = proto.Unmarshal(val, alert)
		return nil
	})

	return
}

// GetAlerts ignores the request and gives all values
func (b *storeImpl) ListAlerts() ([]*v1.ListAlert, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListAlert")

	var alerts []*v1.ListAlert
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(alertListBucket))
		return b.ForEach(func(k, v []byte) error {
			var alert v1.ListAlert
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			alerts = append(alerts, &alert)
			return nil
		})
	})
	return alerts, err
}

// GetAlert returns an alert with given id.
func (b *storeImpl) GetAlert(id string) (alert *v1.Alert, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Alert")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		alert, exists, err = getAlert(id, bucket)
		return err
	})

	return
}

// GetAlerts ignores the request and gives all values
func (b *storeImpl) GetAlerts() ([]*v1.Alert, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Alert")

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
func (b *storeImpl) AddAlert(alert *v1.Alert) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Alert")

	bytes, err := proto.Marshal(alert)
	if err != nil {
		return err
	}

	listBytes, err := marshalAsListAlert(alert)
	if err != nil {
		return err
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		if bucket.Get([]byte(alert.Id)) != nil {
			return fmt.Errorf("Alert %v cannot be added because it already exists", alert.GetId())
		}
		if err := bucket.Put([]byte(alert.Id), bytes); err != nil {
			return err
		}
		bucket = tx.Bucket([]byte(alertListBucket))
		return bucket.Put([]byte(alert.Id), listBytes)
	})
}

// UpdateAlert upserts an alert into Bolt
func (b *storeImpl) UpdateAlert(alert *v1.Alert) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Alert")

	bytes, err := proto.Marshal(alert)
	if err != nil {
		return err
	}

	listBytes, err := marshalAsListAlert(alert)
	if err != nil {
		return err
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		if err := bucket.Put([]byte(alert.Id), bytes); err != nil {
			return err
		}
		bucket = tx.Bucket([]byte(alertListBucket))
		return bucket.Put([]byte(alert.Id), listBytes)
	})
}

func getAlert(id string, bucket *bolt.Bucket) (alert *v1.Alert, exists bool, err error) {
	alert = new(v1.Alert)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, alert)
	return
}

func marshalAsListAlert(alert *v1.Alert) ([]byte, error) {
	listAlert := convertAlertsToListAlerts(alert)
	return proto.Marshal(listAlert)
}
