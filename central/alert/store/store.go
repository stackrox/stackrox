package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	alertBucket     = "alerts"
	alertListBucket = "alerts_list"
)

// Store provides storage functionality for alerts.
type Store interface {
	ListAlert(id string) (*v1.ListAlert, bool, error)
	ListAlerts() ([]*v1.ListAlert, error)

	GetAlert(id string) (*v1.Alert, bool, error)
	GetAlerts() ([]*v1.Alert, error)
	CountAlerts() (int, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, alertBucket)
	bolthelper.RegisterBucketOrPanic(db, alertListBucket)
	return &storeImpl{
		DB: db,
	}
}
