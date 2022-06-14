package bolt

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/dberrors"
	ops "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

var (
	authProviderBucket = []byte("authProviders")
)

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) store.Store {
	bolthelper.RegisterBucketOrPanic(db, authProviderBucket)
	return &storeImpl{
		db: db,
	}
}

type storeImpl struct {
	db *bolt.DB
}

func addUniqueCheck(tx *bolt.Tx, authProvider *storage.AuthProvider) error {
	if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
		return errors.Wrap(err, "Could not add AuthProvider due to name validation")
	}
	return nil
}

func updateUniqueCheck(tx *bolt.Tx, authProvider *storage.AuthProvider) error {
	if val, _ := secondarykey.GetCurrentUniqueKey(tx, authProviderBucket, authProvider.GetId()); val != authProvider.GetName() {
		if err := secondarykey.UpdateUniqueKey(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
			return errors.Wrap(err, "Could not update auth provider due to name validation")
		}
	}
	return nil
}

// GetAll retrieves authProviders from bolt
func (s *storeImpl) GetAll(_ context.Context) ([]*storage.AuthProvider, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "AuthProvider")
	var authProviders []*storage.AuthProvider
	err := s.db.View(func(tx *bolt.Tx) error {
		provB := tx.Bucket(authProviderBucket)

		return provB.ForEach(func(k, v []byte) error {
			var authProvider storage.AuthProvider
			if err := proto.Unmarshal(v, &authProvider); err != nil {
				return err
			}

			authProviders = append(authProviders, &authProvider)
			return nil
		})
	})
	return authProviders, err
}

// Exists checks if an auth provider exists
func (s *storeImpl) Exists(_ context.Context, id string) (bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Exists, "AuthProvider")

	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(authProviderBucket).Get([]byte(id)) != nil
		return nil
	})
	return exists, err
}

// Upsert upserts an auth provider into bolt
func (s *storeImpl) Upsert(_ context.Context, authProvider *storage.AuthProvider) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "AuthProvider")

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(authProviderBucket)
		if bolthelper.Exists(bucket, authProvider.GetId()) {
			// If it exists, then we are updating
			if err := updateUniqueCheck(tx, authProvider); err != nil {
				return err
			}
		} else {
			if err := addUniqueCheck(tx, authProvider); err != nil {
				return err
			}
		}
		bytes, err := proto.Marshal(authProvider)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(authProvider.GetId()), bytes)
	})
}

// Delete removes an auth provider from bolt
func (s *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "AuthProvider")

	return s.db.Update(func(tx *bolt.Tx) error {
		ab := tx.Bucket(authProviderBucket)
		key := []byte(id)
		if exists := ab.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Auth Provider", ID: id}
		}
		if err := secondarykey.RemoveUniqueKey(tx, authProviderBucket, id); err != nil {
			return err
		}
		return ab.Delete(key)
	})
}
