package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const multiplierBucket = "multipliers"

// Store provides storage functionality for alerts.
type Store interface {
	GetMultiplier(id string) (*v1.Multiplier, bool, error)
	GetMultipliers() ([]*v1.Multiplier, error)
	AddMultiplier(multiplier *v1.Multiplier) (string, error)
	UpdateMultiplier(multiplier *v1.Multiplier) error
	RemoveMultiplier(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, multiplierBucket)
	return &storeImpl{
		DB: db,
	}
}
