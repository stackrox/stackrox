package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dberrors"
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

// UpsertSecret adds or updates the secret in the db.
func (s *storeImpl) UpsertSecret(secret *v1.Secret) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Secret")

	return s.db.Update(func(tx *bolt.Tx) error {
		return writeSecret(tx, secret)
	})
}

// RemoveSecret removes a secret
func (s *storeImpl) RemoveSecret(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Secret")
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(secretBucket))
		key := []byte(id)
		if exists := bucket.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Secret", ID: string(key)}
		}
		return bucket.Delete(key)
	})
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
