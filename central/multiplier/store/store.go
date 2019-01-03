package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var multiplierBucket = []byte("multipliers")

// Store provides storage functionality for alerts.
type Store interface {
	GetMultiplier(id string) (*storage.Multiplier, bool, error)
	GetMultipliers() ([]*storage.Multiplier, error)
	AddMultiplier(multiplier *storage.Multiplier) (string, error)
	UpdateMultiplier(multiplier *storage.Multiplier) error
	RemoveMultiplier(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, multiplierBucket)
	return &storeImpl{
		DB: db,
	}
}
