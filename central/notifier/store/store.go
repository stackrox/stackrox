package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const notifierBucket = "notifiers"

// Store provides storage functionality for alerts.
//go:generate mockery -name=Store
type Store interface {
	GetNotifier(id string) (*v1.Notifier, bool, error)
	GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error)
	AddNotifier(notifier *v1.Notifier) (string, error)
	UpdateNotifier(notifier *v1.Notifier) error
	RemoveNotifier(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, notifierBucket)
	return &storeImpl{
		DB: db,
	}
}
