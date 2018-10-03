package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) upsertListAlert(bucket *bolt.Bucket, alert *v1.Alert) error {
	listAlert := convertAlertsToListAlerts(alert)
	bytes, err := proto.Marshal(listAlert)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(alert.Id), bytes)
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

func convertAlertsToListAlerts(a *v1.Alert) *v1.ListAlert {
	return &v1.ListAlert{
		Id:             a.GetId(),
		Time:           a.GetTime(),
		Stale:          a.GetStale(),
		MarkedStale:    a.GetMarkedStale(),
		LifecycleStage: a.GetLifecycleStage(),
		Policy: &v1.ListAlertPolicy{
			Id:          a.GetPolicy().GetId(),
			Name:        a.GetPolicy().GetName(),
			Severity:    a.GetPolicy().GetSeverity(),
			Description: a.GetPolicy().GetDescription(),
			Categories:  a.GetPolicy().GetCategories(),
		},
		Deployment: &v1.ListAlertDeployment{
			Id:          a.GetDeployment().GetId(),
			Name:        a.GetDeployment().GetName(),
			UpdatedAt:   a.GetDeployment().GetUpdatedAt(),
			ClusterName: a.GetDeployment().GetClusterName(),
			Namespace:   a.GetDeployment().GetNamespace(),
		},
	}
}

func (b *storeImpl) getAlert(id string, bucket *bolt.Bucket) (alert *v1.Alert, exists bool, err error) {
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
func (b *storeImpl) GetAlert(id string) (alert *v1.Alert, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Alert")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		alert, exists, err = b.getAlert(id, bucket)
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
		if err := bucket.Put([]byte(alert.Id), bytes); err != nil {
			return err
		}
		return b.upsertListAlert(tx.Bucket([]byte(alertListBucket)), alert)

	})
}

// UpdateAlert upserts an alert into Bolt
func (b *storeImpl) UpdateAlert(alert *v1.Alert) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Alert")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(alertBucket))
		bytes, err := proto.Marshal(alert)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(alert.Id), bytes); err != nil {
			return err
		}
		return b.upsertListAlert(tx.Bucket([]byte(alertListBucket)), alert)
	})
}
