// This file was originally generated with
// //go:generate cp ../../../../central/policy/store/boltdb/store_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB
}

// GetAll retrieves policies matching the request from bolt
func (s *storeImpl) GetAll(_ context.Context) ([]*storage.Policy, error) {
	var policies []*storage.Policy
	err := s.View(func(tx *bolt.Tx) error {
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

// Upsert updates a policy to bolt
func (s *storeImpl) Upsert(_ context.Context, policy *storage.Policy) error {
	return s.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		// If the update is changing the name, check if the name has already been taken
		val, ok := secondarykey.GetCurrentUniqueKey(tx, policyBucket, policy.GetId())
		if !ok {
			if err := secondarykey.InsertUniqueKey(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
				return err
			}
		} else if val != policy.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
				return errors.Wrap(err, "Could not update policy due to name validation")
			}
		}

		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(policy.GetId()), bytes)
	})
}
