package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var processWhitelistBucket = []byte("processWhitelists")

// Store provides storage functionality for process whitelists
//go:generate mockgen-wrapper Store
type Store interface {
	GetWhitelist(id string) (*storage.ProcessWhitelist, error)
	AddWhitelist(whitelist *storage.ProcessWhitelist) error
	GetWhitelists() ([]*storage.ProcessWhitelist, error)
	UpdateWhitelist(whitelist *storage.ProcessWhitelist) error
	DeleteWhitelist(id string) (bool, error)
}

// New Returns a new instance of Store using a bolt DB
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processWhitelistBucket)
	return &storeImpl{
		DB: db,
	}
}
