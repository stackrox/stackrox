package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const authProviderBucket = "authProviders"

func (b *BoltDB) getAuthProvider(id string, bucket *bolt.Bucket) (authProvider *v1.AuthProvider, exists bool, err error) {
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
func (b *BoltDB) GetAuthProvider(id string) (authProvider *v1.AuthProvider, exists bool, err error) {
	authProvider = new(v1.AuthProvider)
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(authProviderBucket))
		authProvider, exists, err = b.getAuthProvider(id, bucket)
		return err
	})
	return
}

// GetAuthProviders retrieves authProviders from bolt
func (b *BoltDB) GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error) {
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
func (b *BoltDB) AddAuthProvider(authProvider *v1.AuthProvider) (string, error) {
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
		if err := checkUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
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
func (b *BoltDB) UpdateAuthProvider(authProvider *v1.AuthProvider) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(authProviderBucket))
		// If the update is changing the name, check if the name has already been taken
		if getCurrentUniqueKey(tx, authProviderBucket, authProvider.GetId()) != authProvider.GetName() {
			if err := checkUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
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
func (b *BoltDB) RemoveAuthProvider(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(authProviderBucket))
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Auth Provider", ID: id}
		}
		if err := removeUniqueKey(tx, authProviderBucket, id); err != nil {
			return err
		}
		return b.Delete(key)
	})
}
