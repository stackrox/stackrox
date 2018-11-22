package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) upsertNetworkPolicy(np *v1.NetworkPolicy) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(networkPolicyBucket))
		bytes, err := proto.Marshal(np)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(np.GetId()), bytes)
	})
}

// GetNetworkPolicy returns network policy with given id.
func (b *storeImpl) GetNetworkPolicy(id string) (np *v1.NetworkPolicy, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkPolicy")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(networkPolicyBucket))
		np = new(v1.NetworkPolicy)
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, np)
	})
	return
}

// GetNetworkPolicies retrieves network policies matching the request from bolt
func (b *storeImpl) GetNetworkPolicies(request *v1.GetNetworkPoliciesRequest) ([]*v1.NetworkPolicy, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "NetworkPolicy")
	var policies []*v1.NetworkPolicy
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(networkPolicyBucket))
		return bucket.ForEach(func(k, v []byte) error {
			var np v1.NetworkPolicy
			if err := proto.Unmarshal(v, &np); err != nil {
				return err
			}
			if request.GetClusterId() != "" && np.GetClusterId() != request.GetClusterId() {
				return nil
			}
			policies = append(policies, &np)
			return nil
		})
	})
	return policies, err
}

// CountNetworkPolicies returns the number of network policies.
func (b *storeImpl) CountNetworkPolicies() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "NetworkPolicy")
	err = b.View(func(tx *bolt.Tx) error {
		count = tx.Bucket([]byte(networkPolicyBucket)).Stats().KeyN
		return nil
	})

	return
}

// AddNetworkPolicy adds a network policy to bolt
func (b *storeImpl) AddNetworkPolicy(np *v1.NetworkPolicy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "NetworkPolicy")
	return b.upsertNetworkPolicy(np)
}

// UpdateNetworkPolicy updates a network policy to bolt
func (b *storeImpl) UpdateNetworkPolicy(np *v1.NetworkPolicy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "NetworkPolicy")
	return b.upsertNetworkPolicy(np)
}

// RemoveNetworkPolicy removes a network policy
func (b *storeImpl) RemoveNetworkPolicy(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkPolicy")
	var err error
	b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(networkPolicyBucket))
		return bucket.Delete([]byte(id))
	})
	return err
}
