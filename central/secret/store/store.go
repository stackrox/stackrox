package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	// SecretBucket is the bucket tht stores secret objects.
	secretBucket     = "secrets"
	secretListBucket = "secrets_list"
)

// Store provides access and update functions for secrets.
//go:generate mockery -name=Store
type Store interface {
	ListSecrets(id []string) ([]*v1.ListSecret, error)

	CountSecrets() (int, error)
	GetAllSecrets() ([]*v1.Secret, error)
	GetSecret(id string) (*v1.Secret, bool, error)
	UpsertSecret(secret *v1.Secret) error
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
