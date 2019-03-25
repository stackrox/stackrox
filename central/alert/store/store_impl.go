package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	*bolt.DB
}

// GetAlert returns an alert with given id.
func (b *storeImpl) ListAlert(id string) (alert *storage.ListAlert, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ListAlert")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertListBucket)
		alert = new(storage.ListAlert)
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

// GetAlertStates returns a minimal message in order to determine the state of the alerts
func (b *storeImpl) GetAlertStates() ([]*storage.AlertState, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "GetAlertStates")

	var alerts []*storage.AlertState
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(alertListBucket)
		return b.ForEach(func(k, v []byte) error {
			var alert storage.AlertState
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			alerts = append(alerts, &alert)
			return nil
		})
	})
	return alerts, err
}

// ListAlerts returns a minimal form of the Alert struct for faster marshalling
func (b *storeImpl) ListAlerts() ([]*storage.ListAlert, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListAlert")

	var alerts []*storage.ListAlert
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(alertListBucket)
		return b.ForEach(func(k, v []byte) error {
			var alert storage.ListAlert
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
func (b *storeImpl) GetAlert(id string) (alert *storage.Alert, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Alert")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(alertBucket)
		alert, exists, err = getAlert(id, bucket)
		return err
	})

	return
}

func (b *storeImpl) getAllAlerts() ([]*storage.Alert, error) {
	var alerts []*storage.Alert
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(alertBucket)
		return b.ForEach(func(k, v []byte) error {
			var alert storage.Alert
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			alerts = append(alerts, &alert)
			return nil
		})
	})
	return alerts, err
}

// GetAlerts takes in optional ids to filter the request by. No IDs will result in all IDs being returned
func (b *storeImpl) GetAlerts(ids ...string) ([]*storage.Alert, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Alert")

	if len(ids) == 0 {
		return b.getAllAlerts()
	}

	var alerts []*storage.Alert
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(alertBucket)
		for _, id := range ids {
			v := b.Get([]byte(id))
			var alert storage.Alert
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			alerts = append(alerts, &alert)
		}
		return nil
	})

	return alerts, err
}

// AddAlert adds an alert into Bolt
func (b *storeImpl) AddAlert(alert *storage.Alert) error {
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
		bucket := tx.Bucket(alertBucket)
		if bucket.Get([]byte(alert.Id)) != nil {
			return fmt.Errorf("Alert %v cannot be added because it already exists", alert.GetId())
		}
		if err := bucket.Put([]byte(alert.Id), bytes); err != nil {
			return err
		}
		bucket = tx.Bucket(alertListBucket)
		return bucket.Put([]byte(alert.Id), listBytes)
	})
}

// UpdateAlert upserts an alert into Bolt
func (b *storeImpl) UpdateAlert(alert *storage.Alert) error {
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
		bucket := tx.Bucket(alertBucket)
		if err := bucket.Put([]byte(alert.Id), bytes); err != nil {
			return err
		}
		bucket = tx.Bucket(alertListBucket)
		return bucket.Put([]byte(alert.Id), listBytes)
	})
}

func getAlert(id string, bucket *bolt.Bucket) (alert *storage.Alert, exists bool, err error) {
	alert = new(storage.Alert)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, alert)
	return
}

func marshalAsListAlert(alert *storage.Alert) ([]byte, error) {
	listAlert := convertAlertsToListAlerts(alert)
	return proto.Marshal(listAlert)
}
