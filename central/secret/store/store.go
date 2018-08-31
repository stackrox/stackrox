package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	// SecretBucket is the bucket tht stores secret objects.
	secretBucket = "secrets"
)

// Store provides access and update functions for secrets.
//go:generate mockery -name=Store
type Store interface {
	GetAllSecrets() ([]*v1.Secret, error)
	GetSecret(id string) (*v1.Secret, bool, error)
	GetSecretsBatch(ids []string) ([]*v1.Secret, error)
	UpsertSecret(secret *v1.Secret) error
	RemoveSecret(id string) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, secretBucket)
	return &storeImpl{
		db: db,
	}
}
