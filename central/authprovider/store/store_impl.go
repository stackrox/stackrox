package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
)

type storeImpl struct {
	*bolt.DB
}

// GetAuthProviders retrieves authProviders from bolt
func (b *storeImpl) GetAllAuthProviders() ([]*storage.AuthProvider, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "AuthProvider")

	var authProviders []*storage.AuthProvider
	err := b.View(func(tx *bolt.Tx) error {
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

// AddAuthProvider adds an auth provider into bolt
func (b *storeImpl) AddAuthProvider(authProvider *storage.AuthProvider) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "AuthProvider")

	if authProvider.GetId() == "" || authProvider.GetName() == "" {
		return errors.New("auth provider is missing required fields")
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(authProviderBucket)
		if bolthelper.Exists(bucket, authProvider.GetId()) {
			return fmt.Errorf("AuthProvider %v (%v) cannot be added because it already exists", authProvider.GetId(), authProvider.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
			return errors.Wrap(err, "Could not add AuthProvider due to name validation")
		}
		bytes, err := proto.Marshal(authProvider)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(authProvider.GetId()), bytes)
	})
}

// UpdateAuthProvider upserts an auth provider into bolt
func (b *storeImpl) UpdateAuthProvider(authProvider *storage.AuthProvider) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "AuthProvider")

	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(authProviderBucket)
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, authProviderBucket, authProvider.GetId()); val != authProvider.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
				return errors.Wrap(err, "Could not update auth provider due to name validation")
			}
		}
		bytes, err := proto.Marshal(authProvider)
		if err != nil {
			return err
		}
		return b.Put([]byte(authProvider.GetId()), bytes)
	})
}

// RemoveAuthProvider removes an auth provider from bolt
func (b *storeImpl) RemoveAuthProvider(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "AuthProvider")

	return b.Update(func(tx *bolt.Tx) error {
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
