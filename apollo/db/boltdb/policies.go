package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const policyBucket = "policies"

func (b *BoltDB) getPolicy(name string, bucket *bolt.Bucket) (policy *v1.Policy, exists bool, err error) {
	policy = new(v1.Policy)
	val := bucket.Get([]byte(name))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, policy)
	return
}

// GetPolicy returns policy with given id.
func (b *BoltDB) GetPolicy(name string) (policy *v1.Policy, exists bool, err error) {
	policy = new(v1.Policy)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		val := b.Get([]byte(name))
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

// AddPolicy adds a policy to bolt
func (b *BoltDB) AddPolicy(policy *v1.Policy) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(policyBucket))
		_, exists, err := b.getPolicy(policy.Name, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Policy %v cannot be added because it already exists", policy.GetName())
		}
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(policy.Name), bytes)
	})
}

// UpdatePolicy updates a policy to bolt
func (b *BoltDB) UpdatePolicy(policy *v1.Policy) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return b.Put([]byte(policy.Name), bytes)
	})
}

// RemovePolicy removes a policy.
func (b *BoltDB) RemovePolicy(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(policyBucket))
		return b.Delete([]byte(name))
	})
}
