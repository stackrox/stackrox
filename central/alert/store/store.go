package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	alertBucket     = []byte("alerts")
	alertListBucket = []byte("alerts_list")
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper Store
type Store interface {
	ListAlert(id string) (*storage.ListAlert, bool, error)
	ListAlerts() ([]*storage.ListAlert, error)

	GetAlert(id string) (*storage.Alert, bool, error)
	GetAlerts() ([]*storage.Alert, error)
	AddAlert(alert *storage.Alert) error
	UpdateAlert(alert *storage.Alert) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, alertBucket)
	bolthelper.RegisterBucketOrPanic(db, alertListBucket)

	return &storeImpl{
		DB: db,
	}
}
