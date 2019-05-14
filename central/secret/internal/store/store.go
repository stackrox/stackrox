package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	// SecretBucket is the bucket tht stores secret objects.
	secretBucket     = []byte("secrets")
	secretListBucket = []byte("secrets_list")
)

// Store provides access and update functions for secrets.
//go:generate mockgen-wrapper Store
type Store interface {
	ListSecrets(id []string) ([]*storage.ListSecret, error)
	ListAllSecrets() ([]*storage.ListSecret, error)

	CountSecrets() (int, error)
	GetAllSecrets() ([]*storage.Secret, error)
	GetSecret(id string) (*storage.Secret, bool, error)
	UpsertSecret(secret *storage.Secret) error
	RemoveSecret(id string) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, secretBucket)
	bolthelper.RegisterBucketOrPanic(db, secretListBucket)
	return &storeImpl{
		db: db,
	}
}
