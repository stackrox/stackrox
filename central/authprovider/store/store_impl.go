package store

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/dberrors"
	"bitbucket.org/stack-rox/apollo/pkg/secondarykey"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
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
	return
}

// GetAuthProvider returns authProvider with given id.
func (b *storeImpl) GetAuthProvider(id string) (authProvider *v1.AuthProvider, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "AuthProvider")
	authProvider = new(v1.AuthProvider)
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(authProviderBucket))
		authProvider, exists, err = b.getAuthProvider(id, bucket)
		return err
	})
	return
}

// GetAuthProviders retrieves authProviders from bolt
func (b *storeImpl) GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "AuthProvider")
	var authProviders []*v1.AuthProvider
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(authProviderBucket))
		return b.ForEach(func(k, v []byte) error {
			var authProvider v1.AuthProvider
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
func (b *storeImpl) AddAuthProvider(authProvider *v1.AuthProvider) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "AuthProvider")
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Update", "AuthProvider")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(authProviderBucket))
		// If the update is changing the name, check if the name has already been taken
		if secondarykey.GetCurrentUniqueKey(tx, authProviderBucket, authProvider.GetId()) != authProvider.GetName() {
			if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "AuthProvider")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(authProviderBucket))
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Auth Provider", ID: id}
		}
		if err := secondarykey.RemoveUniqueKey(tx, authProviderBucket, id); err != nil {
			return err
		}
		return b.Delete(key)
	})
}
