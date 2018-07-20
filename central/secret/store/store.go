package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const (
	// SecretBucket is the bucket tht stores secret objects.
	secretBucket = "secrets"
	// SecretRelationshipsBucket is the bucket tht stores secret relationship objects.
	secretRelationshipsBucket = "secret_relationships"
)

// Store provides access and update functions for secrets.
//go:generate mockery -name=Store
type Store interface {
	GetAllSecrets() ([]*v1.Secret, error)
	GetSecret(id string) (*v1.Secret, bool, error)
	GetSecretsBatch(ids []string) ([]*v1.Secret, error)
	UpsertSecret(secret *v1.Secret) error

	GetRelationship(id string) (*v1.SecretRelationship, bool, error)
	GetRelationshipBatch(ids []string) ([]*v1.SecretRelationship, error)
	UpsertRelationship(relationship *v1.SecretRelationship) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, secretBucket)
	bolthelper.RegisterBucket(db, secretRelationshipsBucket)
	return &storeImpl{
		db: db,
	}
}
