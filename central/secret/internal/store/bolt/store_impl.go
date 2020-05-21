package bolt

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	// SecretBucket is the bucket tht stores secret objects.
	secretBucket = []byte("secrets")
)

type storeImpl struct {
	db *bolt.DB
}

// New returns an new Store instance on top of the input DB.
func New(db *bolt.DB) store.Store {
	bolthelper.RegisterBucketOrPanic(db, secretBucket)
	return &storeImpl{
		db: db,
	}
}

// CountSecrets returns the number secrets in the secret bucket
func (s *storeImpl) Count() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "Secret")
	err = s.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(secretBucket).Stats().KeyN
		return nil
	})
	return
}

// GetSecret returns the secret for the given id.
func (s *storeImpl) Get(id string) (secret *storage.Secret, exists bool, err error) {
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

func (s *storeImpl) GetMany(ids []string) ([]*storage.Secret, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Secret")
	secrets := make([]*storage.Secret, 0, len(ids))
	var missingIndices []int
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(secretBucket)
		for i, id := range ids {
			v := bucket.Get([]byte(id))
			if v == nil {
				missingIndices = append(missingIndices, i)
				continue
			}
			var secret storage.Secret
			if err := proto.Unmarshal(v, &secret); err != nil {
				return err
			}
			secrets = append(secrets, &secret)
		}
		return nil
	})
	return secrets, missingIndices, err
}

// UpsertSecret adds or updates the secret in the db.
func (s *storeImpl) Upsert(secret *storage.Secret) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Secret")

	return s.db.Update(func(tx *bolt.Tx) error {
		if err := writeSecret(tx, secret); err != nil {
			return err
		}
		return nil
	})
}

// RemoveSecret removes a secret
func (s *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Secret")
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(secretBucket)
		key := []byte(id)
		if exists := bucket.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Secret", ID: string(key)}
		}
		if err := bucket.Delete(key); err != nil {
			return err
		}
		return nil
	})
}

func (s *storeImpl) Walk(fn func(secret *storage.Secret) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Secret")
	return s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(secretBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var secret storage.Secret
			if err := proto.Unmarshal(v, &secret); err != nil {
				return err
			}
			return fn(&secret)
		})
	})
}

// HasSecret returns whether a secret exists for the given id.
func hasSecret(tx *bolt.Tx, id string) bool {
	bucket := tx.Bucket(secretBucket)

	bytes := bucket.Get([]byte(id))
	return bytes != nil
}

// readSecret reads a secret within a transaction.
func readSecret(tx *bolt.Tx, id string) (secret *storage.Secret, err error) {
	bucket := tx.Bucket(secretBucket)

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
	bucket := tx.Bucket(secretBucket)

	bytes, err := proto.Marshal(secret)
	if err != nil {
		return
	}
	return bucket.Put([]byte(secret.GetId()), bytes)
}
