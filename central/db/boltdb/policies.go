package boltdb

import (
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const policyBucket = "policies"

func (b *BoltDB) getPolicy(id string, bucket *bolt.Bucket) (policy *v1.Policy, exists bool, err error) {
	policy = new(v1.Policy)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, policy)
	return
}

// GetPolicy returns policy with given id.
func (b *BoltDB) GetPolicy(id string) (policy *v1.Policy, exists bool, err error) {
	policy = new(v1.Policy)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
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
func (b *BoltDB) GetPolicies(request *v1.GetPoliciesRequest) ([]*v1.Policy, error) {
	var policies []*v1.Policy
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		return b.ForEach(func(k, v []byte) error {
			var policy v1.Policy
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
func (b *BoltDB) AddPolicy(policy *v1.Policy) (string, error) {
	policy.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(policyBucket))
		_, exists, err := b.getPolicy(policy.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Policy %v (%v) cannot be added because it already exists", policy.GetName(), policy.GetId())
		}
		if err := checkUniqueKeyExistsAndInsert(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
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
func (b *BoltDB) UpdatePolicy(policy *v1.Policy) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		// If the update is changing the name, check if the name has already been taken
		if getCurrentUniqueKey(tx, policyBucket, policy.GetId()) != policy.GetName() {
			if err := checkUniqueKeyExistsAndInsert(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
				return fmt.Errorf("Could not update policy due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return b.Put([]byte(policy.GetId()), bytes)
	})
}

// RemovePolicy removes a policy.
func (b *BoltDB) RemovePolicy(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Policy", ID: string(key)}
		}
		if err := removeUniqueKey(tx, policyBucket, id); err != nil {
			return err
		}
		return b.Delete(key)
	})
}

// RenamePolicyCategory renames all occurrence of a policy category to the new requested category.
func (b *BoltDB) RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		return b.ForEach(func(k, v []byte) error {
			var policy v1.Policy
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
func (b *BoltDB) DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		return b.ForEach(func(k, v []byte) error {
			var policy v1.Policy
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
