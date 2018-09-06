package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/uuid"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getAuthProvider(id string, bucket *bolt.Bucket) (authProvider *v1.AuthProvider, exists bool, err error) {
	authProvider = new(v1.AuthProvider)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, authProvider)
	if err != nil {
		return
	}
	return
}

// GetAuthProvider returns authProvider with given id.
func (b *storeImpl) GetAuthProvider(id string) (authProvider *v1.AuthProvider, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "AuthProvider")

	authProvider = new(v1.AuthProvider)
	err = b.View(func(tx *bolt.Tx) error {
		proB := tx.Bucket([]byte(authProviderBucket))
		valB := tx.Bucket([]byte(authValidatedBucket))

		authProvider, exists, err = b.getAuthProvider(id, proB)

		// load whether or not it has been validated.
		val := valB.Get([]byte(id))
		if val != nil {
			authProvider.Validated = true
		}
		return err
	})
	return
}

// GetAuthProviders retrieves authProviders from bolt
func (b *storeImpl) GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "AuthProvider")

	var authProviders []*v1.AuthProvider
	err := b.View(func(tx *bolt.Tx) error {
		provB := tx.Bucket([]byte(authProviderBucket))
		valB := tx.Bucket([]byte(authValidatedBucket))

		return provB.ForEach(func(k, v []byte) error {
			var authProvider v1.AuthProvider
			if err := proto.Unmarshal(v, &authProvider); err != nil {
				return err
			}

			// Make sure to only return matching results.
			if request.GetName() != "" && request.GetName() != authProvider.GetName() {
				return nil
			}
			if request.GetType() != "" && request.GetType() != authProvider.GetType() {
				return nil
			}

			// load whether or not it has been validated.
			val := valB.Get(k)
			if val != nil {
				authProvider.Validated = true
			}

			authProviders = append(authProviders, &authProvider)
			return nil
		})
	})
	return authProviders, err
}

// AddAuthProvider adds an auth provider into bolt
func (b *storeImpl) AddAuthProvider(authProvider *v1.AuthProvider) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "AuthProvider")

	authProvider.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(authProviderBucket))
		_, exists, err := b.getAuthProvider(authProvider.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("AuthProvider %v (%v) cannot be added because it already exists", authProvider.GetId(), authProvider.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
			return fmt.Errorf("Could not add AuthProvider due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(authProvider)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(authProvider.GetId()), bytes)
	})
	return authProvider.Id, err
}

// UpdateAuthProvider upserts an auth provider into bolt
func (b *storeImpl) UpdateAuthProvider(authProvider *v1.AuthProvider) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "AuthProvider")

	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(authProviderBucket))
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, authProviderBucket, authProvider.GetId()); val != authProvider.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
				return fmt.Errorf("Could not update auth provider due to name validation: %s", err)
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
		ab := tx.Bucket([]byte(authProviderBucket))
		key := []byte(id)
		if exists := ab.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Auth Provider", ID: id}
		}
		if err := secondarykey.RemoveUniqueKey(tx, authProviderBucket, id); err != nil {
			return err
		}
		if err := ab.Delete(key); err != nil {
			return err
		}

		vb := tx.Bucket([]byte(authValidatedBucket))
		return vb.Delete(key)
	})
}

// RecordAuthSuccess adds an entry in the validated bucket for the provider, which indicates the provider
// has been successfully used at least once previously.
func (b *storeImpl) RecordAuthSuccess(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "AuthValidated")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(authValidatedBucket))

		timestamp := ptypes.TimestampNow()
		bytes, err := proto.Marshal(timestamp)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(id), bytes)
	})
}
