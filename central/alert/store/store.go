package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const alertBucket = "alerts"

// Store provides storage functionality for alerts.
type Store interface {
	GetAlert(id string) (*v1.Alert, bool, error)
	GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error)
	CountAlerts() (int, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
	RemoveAlert(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, alertBucket)
	return &storeImpl{
		DB: db,
	}
}
