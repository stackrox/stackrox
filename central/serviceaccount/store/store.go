package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	// ServiceAccountBucket is the bucket that stores service account objects
	serviceAccountBucket = []byte("service_accounts")
)

// Store provides access and update functions for service accounts.
//go:generate mockgen-wrapper Store
type Store interface {
	ListServiceAccounts(id []string) ([]*storage.ServiceAccount, error)

	CountServiceAccounts() (int, error)
	GetAllServiceAccounts() ([]*storage.ServiceAccount, error)
	GetServiceAccount(id string) (*storage.ServiceAccount, bool, error)
	UpsertServiceAccount(sa *storage.ServiceAccount) error
	RemoveServiceAccount(id string) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, serviceAccountBucket)
	return &storeImpl{
		db: db,
	}
}
