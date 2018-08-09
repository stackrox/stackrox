package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
)

type storeImpl struct {
	db *bolt.DB
}

// GetAllSecrets returns all secrets in the given db.
func (s *storeImpl) GetAllSecrets() (secrets []*v1.Secret, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Secrets")

	s.db.View(func(tx *bolt.Tx) error {
		secrets, err = readAllSecrets(tx)
		return err
	})
	return secrets, err
}

// GetRelationship returns the relationship for the given id.
func (s *storeImpl) GetRelationship(id string) (relationships *v1.SecretRelationship, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "SecretRelationships")

	err = s.db.View(func(tx *bolt.Tx) error {
		if exists = hasRelationship(tx, id); !exists {
			return nil
		}
		relationships, err = readRelationship(tx, id)
		return err
	})
	return
}

// GetRelationshipBatch returns the relationships for the given ids.
func (s *storeImpl) GetRelationshipBatch(ids []string) ([]*v1.SecretRelationship, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "SecretRelationships")

	var relationships []*v1.SecretRelationship
	err := s.db.View(func(tx *bolt.Tx) error {
		for _, id := range ids {
			relationship, err := readRelationship(tx, id)
			if err != nil {
				return err
			}
			relationships = append(relationships, relationship)
		}
		return nil
	})
	return relationships, err
}

// GetSecret returns the secret for the given id.
func (s *storeImpl) GetSecret(id string) (secret *v1.Secret, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Secret")

	err = s.db.View(func(tx *bolt.Tx) error {
		if exists = hasSecret(tx, id); !exists {
			return nil
		}
		secret, err = readSecret(tx, id)
		return err
	})
	return
}

// GetSecretsBatch returns the secrets for the given ids.
func (s *storeImpl) GetSecretsBatch(ids []string) ([]*v1.Secret, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Secrets")

	var secrets []*v1.Secret
	err := s.db.View(func(tx *bolt.Tx) error {
		for _, id := range ids {
			secret, err := readSecret(tx, id)
			if err != nil {
				return err
			}
			secrets = append(secrets, secret)
		}
		return nil
	})
	return secrets, err
}

// UpsertRelationship updates or sets the relationship in bolt.
func (s *storeImpl) UpsertRelationship(relationship *v1.SecretRelationship) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "SecretRelationship")

	return s.db.Update(func(tx *bolt.Tx) error {
		return writeRelationship(tx, relationship)
	})
}

// UpsertSecret adds or updates the secret in the db.
func (s *storeImpl) UpsertSecret(secret *v1.Secret) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Secret")

	return s.db.Update(func(tx *bolt.Tx) error {
		return writeSecret(tx, secret)
	})
}

// hasRelationship returns whether a relatinoship exists for the given id.
func hasRelationship(tx *bolt.Tx, id string) bool {
	bucket := tx.Bucket([]byte(secretRelationshipsBucket))

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		return false
	}
	return true
}

// HasSecret returns whether a secret exists for the given id.
func hasSecret(tx *bolt.Tx, id string) bool {
	bucket := tx.Bucket([]byte(secretBucket))

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		return false
	}
	return true
}

// readAllSecrets reads all the secrets in the DB within a transaction.
func readAllSecrets(tx *bolt.Tx) (secrets []*v1.Secret, err error) {
	bucket := tx.Bucket([]byte(secretBucket))
	err = bucket.ForEach(func(k, v []byte) error {
		secret := new(v1.Secret)
		err = proto.Unmarshal(v, secret)
		if err != nil {
			return err
		}
		secrets = append(secrets, secret)
		return nil
	})
	return
}

// readRelationship reads a raltionship within a transaction.
func readRelationship(tx *bolt.Tx, id string) (relationship *v1.SecretRelationship, err error) {
	bucket := tx.Bucket([]byte(secretRelationshipsBucket))

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("secret relationships with id: %s does not exist", id)
		return
	}

	relationship = new(v1.SecretRelationship)
	err = proto.Unmarshal(bytes, relationship)
	return
}

// readSecret reads a secret within a transaction.
func readSecret(tx *bolt.Tx, id string) (secret *v1.Secret, err error) {
	bucket := tx.Bucket([]byte(secretBucket))

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("secret with id: %s does not exist", id)
		return
	}

	secret = new(v1.Secret)
	err = proto.Unmarshal(bytes, secret)
	return
}

// writeRelationship writes a relationship within a transaction.
func writeRelationship(tx *bolt.Tx, relationship *v1.SecretRelationship) (err error) {
	bucket := tx.Bucket([]byte(secretRelationshipsBucket))

	bytes, err := proto.Marshal(relationship)
	if err != nil {
		return
	}
	bucket.Put([]byte(relationship.GetId()), bytes)
	return
}

// writeSecret writes a secret within a transaction.
func writeSecret(tx *bolt.Tx, secret *v1.Secret) (err error) {
	bucket := tx.Bucket([]byte(secretBucket))

	bytes, err := proto.Marshal(secret)
	if err != nil {
		return
	}
	bucket.Put([]byte(secret.GetId()), bytes)
	return
}
