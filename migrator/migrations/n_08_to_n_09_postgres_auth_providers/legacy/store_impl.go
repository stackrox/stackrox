package store

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

var (
	authProviderBucket = []byte("authProviders")
)

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
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
	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(authProviderBucket).Get([]byte(id)) != nil
		return nil
	})
	return exists, err
}

// Upsert upserts an auth provider into bolt
func (s *storeImpl) Upsert(_ context.Context, authProvider *storage.AuthProvider) error {
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
