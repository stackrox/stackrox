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

// CountRoles returns the number of roles in the roles bucket
func (s *storeImpl) CountRoles() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "K8SRole")
	err = s.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(roleBucket).Stats().KeyN
		return nil
	})
	return
}

// ListAllRoles returns all k8s roles in the given db.
func (s *storeImpl) ListAllRoles() (roles []*storage.K8SRole, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "K8SRole")

	err = s.db.View(func(tx *bolt.Tx) error {
		var err error
		roles, err = readAllRoles(tx)
		return err
	})
	return roles, err
}

// GetRole returns the k8s role for the given id.
func (s *storeImpl) GetRole(id string) (role *storage.K8SRole, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "K8SRole")

	err = s.db.View(func(tx *bolt.Tx) error {
		if exists = hasRole(tx, id); !exists {
			return nil
		}
		role, err = readRole(tx, id)
		return err
	})
	return
}

// ListRoles returns a list of k8s roles from the given ids.
func (s *storeImpl) ListRoles(ids []string) ([]*storage.K8SRole, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "K8SRole")
	roles := make([]*storage.K8SRole, 0, len(ids))
	err := s.db.View(func(tx *bolt.Tx) error {
		for _, id := range ids {
			role, err := getRole(tx, id)
			if err != nil {
				return err
			}
			roles = append(roles, role)
		}
		return nil
	})
	return roles, err
}

// UpsertRole adds or updates the k8s role in the db.
func (s *storeImpl) UpsertRole(role *storage.K8SRole) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "K8SRole")
	bytes, err := proto.Marshal(role)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		if err := writeRole(tx, role, bytes); err != nil {
			return err
		}
		return nil
	})
}

// RemoveRole removes a k8s role
func (s *storeImpl) RemoveRole(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "K8SRole")
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(roleBucket)
		key := []byte(id)
		if bucket.Get(key) == nil {
			return dberrors.ErrNotFound{Type: "K8SRole", ID: id}
		}
		return bucket.Delete(key)
	})
}

func getRole(tx *bolt.Tx, id string) (role *storage.K8SRole, err error) {
	bucket := tx.Bucket(roleBucket)
	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("role with id: %q does not exist", id)
		return
	}

	role = new(storage.K8SRole)
	err = proto.Unmarshal(bytes, role)
	return
}

// hasRole returns whether a role exists for the given id.
func hasRole(tx *bolt.Tx, id string) bool {
	bucket := tx.Bucket(roleBucket)

	return bucket.Get([]byte(id)) != nil
}

// readAllRoles reads all the roles in the DB within a transaction.
func readAllRoles(tx *bolt.Tx) (roles []*storage.K8SRole, err error) {
	bucket := tx.Bucket(roleBucket)
	err = bucket.ForEach(func(k, v []byte) error {
		role := new(storage.K8SRole)
		err = proto.Unmarshal(v, role)
		if err != nil {
			return err
		}
		roles = append(roles, role)
		return nil
	})
	return
}

// readRole reads a k8s role within a transaction.
func readRole(tx *bolt.Tx, id string) (role *storage.K8SRole, err error) {
	bucket := tx.Bucket(roleBucket)

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("role with id: %s does not exist", id)
		return
	}

	role = new(storage.K8SRole)
	err = proto.Unmarshal(bytes, role)
	return
}

// writeRole writes a k8s role within a transaction.
func writeRole(tx *bolt.Tx, role *storage.K8SRole, bytes []byte) (err error) {
	bucket := tx.Bucket(roleBucket)
	return bucket.Put([]byte(role.GetId()), bytes)
}
