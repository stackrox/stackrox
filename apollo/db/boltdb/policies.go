package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const policyBucket = "policies"

// GetPolicy returns a policy with given name.
func (b *BoltDB) GetPolicy(name string) (policy *v1.Policy, exists bool, err error) {
	policy = new(v1.Policy)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		val := b.Get([]byte(name))
		if val == nil {
			exists = false
			return nil
		}

		exists = true
		return proto.Unmarshal(val, policy)
	})

	return
}

// GetPolicies returns all policies regardless of request.
func (b *BoltDB) GetPolicies(*v1.GetPoliciesRequest) ([]*v1.Policy, error) {
	var policies []*v1.Policy
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		b.ForEach(func(k, v []byte) error {
			var policy v1.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				return err
			}
			policies = append(policies, &policy)
			return nil
		})
		return nil
	})
	return policies, err
}

func (b *BoltDB) upsertPolicy(policy *v1.Policy) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		bytes, err := proto.Marshal(policy)
		if err != nil {
			log.Error(err)
			return err
		}
		return b.Put([]byte(policy.Name), bytes)
	})
}

// AddPolicy inserts the policy.
func (b *BoltDB) AddPolicy(policy *v1.Policy) error {
	return b.upsertPolicy(policy)
}

// UpdatePolicy updates the policy.
func (b *BoltDB) UpdatePolicy(policy *v1.Policy) error {
	return b.upsertPolicy(policy)
}

// RemovePolicy removes the policy from the database.
func (b *BoltDB) RemovePolicy(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		err := b.Delete([]byte(name))
		return err
	})
}
