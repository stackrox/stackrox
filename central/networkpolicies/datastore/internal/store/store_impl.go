package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) upsertNetworkPolicy(np *storage.NetworkPolicy) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		bytes, err := proto.Marshal(np)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(np.GetId()), bytes)
	})
}

// GetNetworkPolicy returns network policy with given id.
func (b *storeImpl) GetNetworkPolicy(id string) (np *storage.NetworkPolicy, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkPolicy")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		np = new(storage.NetworkPolicy)
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, np)
	})
	return
}

func (b *storeImpl) doForMatchingPolicies(clusterID, namespace string, f func(*storage.NetworkPolicy)) error {
	return b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var np storage.NetworkPolicy
			if err := proto.Unmarshal(v, &np); err != nil {
				return err
			}
			if clusterID != "" && np.GetClusterId() != clusterID {
				return nil
			}
			if namespace != "" && np.GetNamespace() != namespace {
				return nil
			}
			f(&np)
			return nil
		})
	})
}

// GetNetworkPolicies retrieves network policies matching the request from bolt
func (b *storeImpl) GetNetworkPolicies(clusterID, namespace string) ([]*storage.NetworkPolicy, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "NetworkPolicy")
	var policies []*storage.NetworkPolicy
	err := b.doForMatchingPolicies(clusterID, namespace, func(np *storage.NetworkPolicy) {
		policies = append(policies, np)
	})
	return policies, err
}

func (b *storeImpl) CountMatchingNetworkPolicies(clusterID, namespace string) (count int, err error) {
	err = b.doForMatchingPolicies(clusterID, namespace, func(_ *storage.NetworkPolicy) {
		count++
	})
	return
}

// AddNetworkPolicy adds a network policy to bolt
func (b *storeImpl) AddNetworkPolicy(np *storage.NetworkPolicy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "NetworkPolicy")
	return b.upsertNetworkPolicy(np)
}

// UpdateNetworkPolicy updates a network policy to bolt
func (b *storeImpl) UpdateNetworkPolicy(np *storage.NetworkPolicy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "NetworkPolicy")
	return b.upsertNetworkPolicy(np)
}

// RemoveNetworkPolicy removes a network policy
func (b *storeImpl) RemoveNetworkPolicy(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "NetworkPolicy")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(networkPolicyBucket)
		return bucket.Delete([]byte(id))
	})
}
