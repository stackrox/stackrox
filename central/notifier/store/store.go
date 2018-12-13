package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const notifierBucket = "notifiers"

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper Store
type Store interface {
	GetNotifier(id string) (*storage.Notifier, bool, error)
	GetNotifiers(request *v1.GetNotifiersRequest) ([]*storage.Notifier, error)
	AddNotifier(notifier *storage.Notifier) (string, error)
	UpdateNotifier(notifier *storage.Notifier) error
	RemoveNotifier(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, notifierBucket)
	return &storeImpl{
		DB: db,
	}
}
