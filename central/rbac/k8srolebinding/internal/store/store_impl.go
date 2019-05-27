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

// ListAllRoles returns all k8s role bindings in the given db.
func (s *storeImpl) ListAllRoleBindings() (bindings []*storage.K8SRoleBinding, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "K8SRoleBinding")

	err = s.db.View(func(tx *bolt.Tx) error {
		var err error
		bindings, err = readAllRoleBindings(tx)
		return err
	})
	return bindings, err
}

// GetRoleBinding returns the k8s role binding for the given id.
func (s *storeImpl) GetRoleBinding(id string) (binding *storage.K8SRoleBinding, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "K8SRoleBinding")

	err = s.db.View(func(tx *bolt.Tx) error {
		if exists = hasRoleBinding(tx, id); !exists {
			return nil
		}
		binding, err = readRoleBinding(tx, id)
		return err
	})
	return
}

// UpsertRoleBinding adds or updates the k8s role binding in the db.
func (s *storeImpl) UpsertRoleBinding(binding *storage.K8SRoleBinding) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "K8SRoleBinding")
	bytes, err := proto.Marshal(binding)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := writeRoleBinding(tx, binding, bytes); err != nil {
			return err
		}
		return nil
	})
}

// RemoteRoleBinding removes a k8s role binding
func (s *storeImpl) RemoveRoleBinding(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "K8SRoleBinding")
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(roleBindingBucket)
		key := []byte(id)
		if bucket.Get(key) == nil {
			return dberrors.ErrNotFound{Type: "K8SRoleBinding", ID: id}
		}
		return bucket.Delete([]byte(id))

	})
}

// hasRoleBinding returns whether a role binding exists for the given id.
func hasRoleBinding(tx *bolt.Tx, id string) bool {
	bucket := tx.Bucket(roleBindingBucket)

	return bucket.Get([]byte(id)) != nil
}

// readAllRoleBindings reads all the role bindings in the DB within a transaction.
func readAllRoleBindings(tx *bolt.Tx) (bindings []*storage.K8SRoleBinding, err error) {
	bucket := tx.Bucket(roleBindingBucket)
	err = bucket.ForEach(func(k, v []byte) error {
		binding := new(storage.K8SRoleBinding)
		err = proto.Unmarshal(v, binding)
		if err != nil {
			return err
		}
		bindings = append(bindings, binding)
		return nil
	})
	return
}

// readRoleBinding reads a k8s role binding within a transaction.
func readRoleBinding(tx *bolt.Tx, id string) (binding *storage.K8SRoleBinding, err error) {
	bucket := tx.Bucket(roleBindingBucket)

	bytes := bucket.Get([]byte(id))
	if bytes == nil {
		err = fmt.Errorf("role binding with id: %q does not exist", id)
		return
	}

	binding = new(storage.K8SRoleBinding)
	err = proto.Unmarshal(bytes, binding)
	return
}

// writeRoleBinding writes a k8s role binding within a transaction.
func writeRoleBinding(tx *bolt.Tx, binding *storage.K8SRoleBinding, bytes []byte) (err error) {
	bucket := tx.Bucket(roleBindingBucket)
	return bucket.Put([]byte(binding.GetId()), bytes)
}
