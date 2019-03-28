package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
)

// Store is a store for license keys.
type Store interface {
	ListLicenseKeys() ([]*storage.StoredLicenseKey, error)
	UpsertLicenseKey(key *storage.StoredLicenseKey) error
	DeleteLicenseKey(licenseID string) error
}

// New creates a new license key store.
func New(db *bolt.DB) (Store, error) {
	return newStore(db)
}

//go:generate mockgen-wrapper Store
