package store

import (
	"fmt"
	"time"

	"github.com/deckarep/golang-set"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	db *bolt.DB
}

func getListSecret(tx *bolt.Tx, id string) (secret *storage.ListSecret, err error) {
	bucket := tx.Bucket([]byte(secretListBucket))
	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("secret with id: %s does not exist", id)
		return
	}

	secret = new(storage.ListSecret)
	err = proto.Unmarshal(bytes, secret)
	return
}

// Note: This is called within a txn and does not require an Update or View
func removeListSecret(tx *bolt.Tx, id string) error {
	bucket := tx.Bucket([]byte(secretListBucket))
	return bucket.Delete([]byte(id))
}

// Note: This is called within a txn and do not require an Update or View
func upsertListSecret(tx *bolt.Tx, secret *storage.Secret) error {
	bucket := tx.Bucket([]byte(secretListBucket))
	listSecret := convertSecretToSecretList(secret)
	bytes, err := proto.Marshal(listSecret)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(secret.Id), bytes)
}

func convertSecretToSecretList(s *storage.Secret) *storage.ListSecret {
	typeSet := mapset.NewSet()
	var typeSlice []storage.SecretType
	for _, f := range s.GetFiles() {
		if !typeSet.Contains(f.GetType()) {
			typeSlice = append(typeSlice, f.GetType())
			typeSet.Add(f.GetType())
		}
	}
	if len(typeSlice) == 0 {
		typeSlice = append(typeSlice, storage.SecretType_UNDETERMINED)
	}

	return &storage.ListSecret{
		Id:          s.GetId(),
		Name:        s.GetName(),
		ClusterName: s.GetClusterName(),
		Namespace:   s.GetNamespace(),
		Types:       typeSlice,
		CreatedAt:   s.GetCreatedAt(),
	}
}

// CountSecrets returns the number secrets in the secret bucket
func (s *storeImpl) CountSecrets() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "Secret")
	err = s.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket([]byte(secretBucket)).Stats().KeyN
		return nil
	})
	return
}

// GetAllSecrets returns all secrets in the given db.
func (s *storeImpl) GetAllSecrets() (secrets []*storage.Secret, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Secret")

	err = s.db.View(func(tx *bolt.Tx) error {
		var err error
		secrets, err = readAllSecrets(tx)
		return err
	})
	return secrets, err
}

func (s *storeImpl) ListAllSecrets() (secrets []*storage.ListSecret, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.List, "Secret")

	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(secretListBucket))
		return bucket.ForEach(func(k, v []byte) error {
			secret := new(storage.ListSecret)
			err := proto.Unmarshal(v, secret)
			if err != nil {
				return err
			}
			secrets = append(secrets, secret)
			return nil

		})
	})
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// GetSecret returns the secret for the given id.
func (s *storeImpl) GetSecret(id string) (secret *storage.Secret, exists bool, err error) {
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

// ListSecrets returns a list of secrets from the given ids.
func (s *storeImpl) ListSecrets(ids []string) ([]*storage.ListSecret, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListSecret")
	secrets := make([]*storage.ListSecret, 0, len(ids))
	err := s.db.View(func(tx *bolt.Tx) error {
		for _, id := range ids {
			secret, err := getListSecret(tx, id)
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
func (s *storeImpl) UpsertSecret(secret *storage.Secret) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Secret")

	return s.db.Update(func(tx *bolt.Tx) error {
		if err := writeSecret(tx, secret); err != nil {
			return err
		}
		return upsertListSecret(tx, secret)
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
		if err := bucket.Delete(key); err != nil {
			return err
		}
		return removeListSecret(tx, id)
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
func readAllSecrets(tx *bolt.Tx) (secrets []*storage.Secret, err error) {
	bucket := tx.Bucket([]byte(secretBucket))
	err = bucket.ForEach(func(k, v []byte) error {
		secret := new(storage.Secret)
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
func readSecret(tx *bolt.Tx, id string) (secret *storage.Secret, err error) {
	bucket := tx.Bucket([]byte(secretBucket))

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("secret with id: %s does not exist", id)
		return
	}

	secret = new(storage.Secret)
	err = proto.Unmarshal(bytes, secret)
	return
}

// writeSecret writes a secret within a transaction.
func writeSecret(tx *bolt.Tx, secret *storage.Secret) (err error) {
	bucket := tx.Bucket([]byte(secretBucket))

	bytes, err := proto.Marshal(secret)
	if err != nil {
		return
	}
	bucket.Put([]byte(secret.GetId()), bytes)
	return
}
