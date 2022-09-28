// This file was originally generated with
// //go:generate cp ../../../../central/policy/store/boltdb/store_impl.go .

package legacy

import (
	"context"
	"encoding/json"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/set"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB
}

func (s *storeImpl) getAllFromStore(_ context.Context) ([]*storage.Policy, error) {
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

// GetAll retrieves policies matching the request from bolt
func (s *storeImpl) GetAll(_ context.Context) ([]*storage.Policy, error) {
	policyIDSet := set.NewStringSet()
	policies, err := s.getAllFromStore(policyCtx)
	if err != nil {
		return nil, err
	}

	for _, p := range policies {
		policyIDSet.Add(p.GetId())
	}

	// Get removed default policies and disable them.
	// Before 3.66, default policies can be removed and the removed default policies are stored in
	// bucket removed_default_policies. Starting from 3.66, we do not support deleting default policies.
	// With Postgres central db, we do not have a dedicated table to store these policies, instead we
	// disable them during migration so that they won't come back.
	disabledPolicies, err := s.getDisabledRemovedDefaultPolicies(policyIDSet)
	if err != nil {
		log.WriteToStderrf("Failed to get default policies %v", err)
	}

	return append(policies, disabledPolicies...), nil
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

func (s *storeImpl) wasDefaultPolicyRemoved(id string) (bool, error) {
	var wasRemoved bool
	if err := s.View(func(tx *bolt.Tx) error {
		val := tx.Bucket(removedDefaultPolicyBucket).Get([]byte(id))
		if val == nil {
			wasRemoved = false
			return nil
		}
		return json.Unmarshal(val, &wasRemoved)
	}); err != nil {
		return false, err
	}
	return wasRemoved, nil
}

func (s *storeImpl) getDisabledRemovedDefaultPolicies(policySet set.StringSet) ([]*storage.Policy, error) {
	errList := errorhelpers.NewErrorList("removed default policies")
	rawPolicies, err := getRawDefaultPolicies()
	if err != nil {
		errList.AddError(err)
	}

	var policies []*storage.Policy
	for _, p := range rawPolicies {
		// Skip exising default policies in primary policy DB.
		// We do not want to change existing policies.
		if policySet.Contains(p.GetId()) {
			continue
		}

		wasRemoved, err := s.wasDefaultPolicyRemoved(p.GetId())
		if err != nil {
			// Add error and skip adding default policy
			errList.AddWrapf(err, "Could not determine if the default policy %s was previously removed, skip it", p.GetId())
			continue
		}
		if !wasRemoved {
			continue
		}

		if err := policyversion.EnsureConvertedTo(p, currentVersion); err != nil {
			errList.AddWrapf(err, "converting policy %s", p.GetName())
			continue
		}
		log.WriteToStderrf("Disabling removed default policy %q", p.GetId())
		p.Disabled = true

		// fill multi-word sort helper field
		policyutils.FillSortHelperFields(p)

		policies = append(policies, p)
	}

	return policies, errList.ToError()
}
