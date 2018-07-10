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
type Store interface {
	GetAllSecrets() (secrets []*v1.Secret, err error)
	GetRelationship(id string) (relationships *v1.SecretRelationship, exists bool, err error)
	GetRelationshipBatch(ids []string) ([]*v1.SecretRelationship, error)
	GetSecret(id string) (secret *v1.Secret, exists bool, err error)
	GetSecretsBatch(ids []string) ([]*v1.Secret, error)

	UpsertRelationship(relationship *v1.SecretRelationship) error
	UpsertSecret(secret *v1.Secret) error
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, secretBucket)
	bolthelper.RegisterBucket(db, secretRelationshipsBucket)
	return &storeImpl{
		db: db,
	}
}
