package store

import (
	"fmt"
	"sort"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/uuid"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getPolicy(id string, bucket *bolt.Bucket) (policy *storage.Policy, exists bool, err error) {
	policy = new(storage.Policy)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, policy)
	return
}

// GetPolicy returns policy with given id.
func (b *storeImpl) GetPolicy(id string) (policy *storage.Policy, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Policy")
	policy = new(storage.Policy)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(policyBucket)
		val := b.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, policy)
	})

	return
}

// GetPolicies retrieves policies matching the request from bolt
func (b *storeImpl) GetPolicies() ([]*storage.Policy, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Policy")
	var policies []*storage.Policy
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(policyBucket)
		return b.ForEach(func(k, v []byte) error {
			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				return err
			}
			policies = append(policies, &policy)
			return nil
		})
	})
	return policies, err
}

// AddPolicy adds a policy to bolt
func (b *storeImpl) AddPolicy(policy *storage.Policy) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Policy")
	if policy.Id == "" {
		policy.Id = uuid.NewV4().String()
	}
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		_, exists, err := b.getPolicy(policy.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Policy %v (%v) cannot be added because it already exists", policy.GetName(), policy.GetId())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
			return fmt.Errorf("Could not add policy due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(policy.GetId()), bytes)
	})
	return policy.Id, err
}

// UpdatePolicy updates a policy to bolt
func (b *storeImpl) UpdatePolicy(policy *storage.Policy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Policy")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, policyBucket, policy.GetId()); val != policy.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
				return fmt.Errorf("Could not update policy due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(policy.GetId()), bytes)
	})
}

// RemovePolicy removes a policy.
func (b *storeImpl) RemovePolicy(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Policy")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		key := []byte(id)
		if exists := bucket.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Policy", ID: string(key)}
		}
		if err := secondarykey.RemoveUniqueKey(tx, policyBucket, id); err != nil {
			return err
		}
		return bucket.Delete(key)
	})
}

// RenamePolicyCategory renames all occurrence of a policy category to the new requested category.
func (b *storeImpl) RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Rename, "PolicyCategory")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(policyBucket)
		return b.ForEach(func(k, v []byte) error {
			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				return err
			}

			modified := false
			for i, c := range policy.GetCategories() {
				if c == request.GetOldCategory() {
					policy.Categories[i] = request.GetNewCategory()
					modified = true
				}
			}

			if modified {
				sort.Strings(policy.Categories)
				bytes, err := proto.Marshal(&policy)
				if err != nil {
					return err
				}
				return b.Put([]byte(policy.GetId()), bytes)
			}
			return nil
		})
	})
}

// DeletePolicyCategory removes a category from all policies.
func (b *storeImpl) DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "PolicyCategory")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(policyBucket)
		return b.ForEach(func(k, v []byte) error {
			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				return err
			}

			removed := policy.GetCategories()[:0]
			for _, c := range policy.GetCategories() {
				if c != request.GetCategory() {
					removed = append(removed, c)
				}
			}

			if len(removed) != len(policy.GetCategories()) {
				policy.Categories = removed
				bytes, err := proto.Marshal(&policy)
				if err != nil {
					return err
				}
				return b.Put([]byte(policy.GetId()), bytes)
			}
			return nil
		})
	})
}
