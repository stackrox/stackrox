package store

import (
	"fmt"
	"time"

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

// hasServiceAccount returns whether a service account exists for the given id.
func hasServiceAccount(tx *bolt.Tx, id string) bool {
	bucket := tx.Bucket(serviceAccountBucket)

	bytes := bucket.Get([]byte(id))
	return bytes != nil
}

// writeServiceAccount writes a service account within a transaction.
func writeServiceAccount(tx *bolt.Tx, sa *storage.ServiceAccount, bytes []byte) (err error) {
	bucket := tx.Bucket(serviceAccountBucket)
	if err != nil {
		return
	}
	return bucket.Put([]byte(sa.GetId()), bytes)
}

// readServiceAccount reads a service account within a transaction.
func readServiceAccount(tx *bolt.Tx, id string) (sa *storage.ServiceAccount, err error) {
	bucket := tx.Bucket(serviceAccountBucket)

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("service account with id: %s does not exist", id)
		return
	}

	sa = new(storage.ServiceAccount)
	err = proto.Unmarshal(bytes, sa)
	return
}

// readAllServiceAccounts reads all the ServiceAccounts in the DB within a transaction.
func readAllServiceAccounts(tx *bolt.Tx) (serviceAccounts []*storage.ServiceAccount, err error) {
	bucket := tx.Bucket(serviceAccountBucket)
	err = bucket.ForEach(func(k, v []byte) error {
		sa := new(storage.ServiceAccount)
		err = proto.Unmarshal(v, sa)
		if err != nil {
			return err
		}
		serviceAccounts = append(serviceAccounts, sa)
		return nil
	})
	return
}

// Note: This is called within a txn and does not require an Update or View
func removeServiceAccount(tx *bolt.Tx, id string) error {
	bucket := tx.Bucket(serviceAccountBucket)
	return bucket.Delete([]byte(id))
}

// Note: This is called within a txn and do not require an Update or View
func upsertServiceAccount(tx *bolt.Tx, sa *storage.ServiceAccount, bytes []byte) error {
	bucket := tx.Bucket(serviceAccountBucket)
	return bucket.Put([]byte(sa.Id), bytes)
}

// GetAllServiceAccounts returns all service accounts in the given db.
func (s *storeImpl) GetAllServiceAccounts() (serviceAccounts []*storage.ServiceAccount, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Service Account")

	err = s.db.View(func(tx *bolt.Tx) error {
		var err error
		serviceAccounts, err = readAllServiceAccounts(tx)
		return err
	})
	return serviceAccounts, err
}

// GetServiceAccount returns the ServiceAccount for the given id.
func (s *storeImpl) GetServiceAccount(id string) (sa *storage.ServiceAccount, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ServiceAccount")

	err = s.db.View(func(tx *bolt.Tx) error {
		if exists = hasServiceAccount(tx, id); !exists {
			return nil
		}
		sa, err = readServiceAccount(tx, id)
		return err
	})
	return
}

// UpsertServiceAccount adds or updates the service account in the db.
func (s *storeImpl) UpsertServiceAccount(sa *storage.ServiceAccount) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "ServiceAccount")
	bytes, err := proto.Marshal(sa)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		if err := writeServiceAccount(tx, sa, bytes); err != nil {
			return err
		}
		return upsertServiceAccount(tx, sa, bytes)
	})
}

// RemoveServiceAccount removes a service account
func (s *storeImpl) RemoveServiceAccount(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ServiceAccount")
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(serviceAccountBucket)
		key := []byte(id)
		if exists := bucket.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "ServiceAccount", ID: string(key)}
		}
		if err := bucket.Delete(key); err != nil {
			return err
		}
		return removeServiceAccount(tx, id)
	})
}
