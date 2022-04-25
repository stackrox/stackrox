package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB

	mutex sync.Mutex
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
		// Do not block adding policy if an error was encountered figuring it out.
		return false, err
	}
	return wasRemoved, nil
}

func (s *storeImpl) getPolicy(id string, bucket *bolt.Bucket) (policy *storage.Policy, exists bool, err error) {
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
func (s *storeImpl) GetPolicy(id string) (policy *storage.Policy, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Policy")
	policy = new(storage.Policy)
	err = s.View(func(tx *bolt.Tx) error {
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

// GetAllPolicies retrieves policies matching the request from bolt
func (s *storeImpl) GetAllPolicies() ([]*storage.Policy, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Policy")
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

func (s *storeImpl) GetPolicies(ids ...string) ([]*storage.Policy, []int, []error, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Policy")
	var policies []*storage.Policy
	var missingIndices []int
	var errorList []error
	err := s.View(func(tx *bolt.Tx) error {
		for i, id := range ids {
			policy := new(storage.Policy)
			b := tx.Bucket(policyBucket)
			val := b.Get([]byte(id))
			if val == nil {
				missingIndices = append(missingIndices, i)
				errorList = append(errorList, errors.New("not found"))
				continue
			}
			if err := proto.Unmarshal(val, policy); err != nil {
				missingIndices = append(missingIndices, i)
				errorList = append(errorList, err)
				continue
			}
			policies = append(policies, policy)
		}
		return nil
	})

	return policies, missingIndices, errorList, err
}

// AddPolicy adds a policy to bolt
func (s *storeImpl) AddPolicy(policy *storage.Policy, removePolicyTombstone bool) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Policy")

	if policy.Id == "" {
		policy.Id = uuid.NewV4().String()
	}

	// Lock here so we can check whether the policy exists outside of the Bolt write lock.  This can race with
	// update/delete but I don't think this results in any problems.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if removePolicyTombstone {
		if err := s.Update(func(tx *bolt.Tx) error {
			return tx.Bucket(removedDefaultPolicyBucket).Delete([]byte(policy.GetId()))
		}); err != nil {
			return "", err
		}
	} else {
		wasRemoved, err := s.wasDefaultPolicyRemoved(policy.GetId())
		if err != nil {
			return "", err
		}
		if wasRemoved {
			return "", errors.Errorf("default policy %s was previously removed", policy.GetId())
		}
	}

	equalPolicyExists := false
	err := s.View(func(tx *bolt.Tx) error {
		var errs []error
		bucket := tx.Bucket(policyBucket)

		// Check whether a policy with this ID already exists
		existingPolicy, exists, err := s.getPolicy(policy.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			if proto.Equal(existingPolicy, policy) {
				// The existing policy is identical to the added policy, this is a noop.
				equalPolicyExists = true
				return nil
			}
			errs = append(errs,
				&IDConflictError{
					ErrString:          fmt.Sprintf("Policy %v (%v) cannot be added because it already exists", policy.GetName(), policy.GetId()),
					ExistingPolicyName: existingPolicy.GetName(),
				},
			)
		}

		// Check whether a policy with this name already exists
		if exists := secondarykey.CheckUniqueKeyExists(tx, policyBucket, policy.GetName()); exists {
			errs = append(errs,
				&NameConflictError{
					ErrString:          "Could not add policy due to name validation",
					ExistingPolicyName: policy.GetName(),
				},
			)
		}

		// If we had any ID or name conflicts return both here
		if len(errs) > 0 {
			return &PolicyStoreErrorList{
				Errors: errs,
			}
		}
		return nil
	})
	if err != nil {
		return policy.GetId(), err
	}

	if equalPolicyExists {
		return policy.GetId(), nil
	}

	if err = s.Update(func(tx *bolt.Tx) error {
		// We've already checked for duplicate IDs and names, we can just write here.
		if err := secondarykey.InsertUniqueKey(tx, policyBucket, policy.GetId(), policy.GetName()); err != nil {
			return err
		}

		bucket := tx.Bucket(policyBucket)
		bytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(policy.GetId()), bytes)
	}); err != nil {
		return "", err
	}

	return policy.GetId(), nil
}

// UpdatePolicy updates a policy to bolt
func (s *storeImpl) UpdatePolicy(policy *storage.Policy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Policy")

	// Have to lock here because this is an upsert, not an update.  AddPolicy should not re-create a policy which is
	// created here.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Verify that policy's settings flags are not updated. If the lock flags are set to read-only,
	// verify that the corresponding fields are not updated.
	if err := s.verifySettingFieldsAreUnchanged(policy); err != nil {
		return err
	}

	err := s.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		// If the update is changing the name, check if the name has already been taken
		val, ok := secondarykey.GetCurrentUniqueKey(tx, policyBucket, policy.GetId())
		if !ok {
			return errox.NotFound
		}
		if val != policy.GetName() {
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
	if err != nil {
		return err
	}

	// If the policy is marked as default, then we prohibit deletion and do not use 'removedDefaultPolicyBucket'.
	if policy.GetIsDefault() {
		return nil
	}

	// All policies upserted after initial addition are treated as custom policies. A default policy may have been
	// exported, then deleted and re-imported. Ensure that it is removed from the bucket.
	return s.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(removedDefaultPolicyBucket).Delete([]byte(policy.GetId()))
	})
}

// RemovePolicy removes a policy.
func (s *storeImpl) RemovePolicy(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Policy")
	var policy storage.Policy
	err := s.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		key := []byte(id)
		val := bucket.Get(key)
		if val == nil {
			return dberrors.ErrNotFound{Type: "Policy", ID: string(key)}
		}

		if err := proto.Unmarshal(val, &policy); err != nil {
			return err
		}

		if policy.GetIsDefault() {
			return errors.Errorf("policy %s is a default policy. Default system policies cannot be removed", id)
		}

		if err := secondarykey.RemoveUniqueKey(tx, policyBucket, id); err != nil {
			return err
		}
		return bucket.Delete(key)
	})
	if err != nil {
		return err
	}

	// If the policy is marked as default, then we prohibit deletion and do not use 'removedDefaultPolicyBucket'.
	if policy.GetIsDefault() {
		return nil
	}

	return s.updatePolicyTombstone(id, true, false)
}

func (s *storeImpl) updatePolicyTombstone(id string, tombstoned bool, addIfNotFound bool) error {
	return s.Update(func(tx *bolt.Tx) error {
		if tx.Bucket(removedDefaultPolicyBucket).Get([]byte(id)) == nil && !addIfNotFound {
			return nil
		}

		bytes, err := json.Marshal(tombstoned)
		if err != nil {
			return err
		}
		return tx.Bucket(removedDefaultPolicyBucket).Put([]byte(id), bytes)
	})
}

// RenamePolicyCategory renames all occurrence of a policy category to the new requested category.
func (s *storeImpl) RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Rename, "PolicyCategory")
	return s.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(policyBucket)
		return b.ForEach(func(k, v []byte) error {
			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				return err
			}

			modified := false
			for i, c := range policy.GetCategories() {
				if c == request.GetOldCategory() {
					policy.LastUpdated = protoconv.ConvertTimeToTimestamp(time.Now())
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
func (s *storeImpl) DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "PolicyCategory")
	return s.Update(func(tx *bolt.Tx) error {
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

func (s *storeImpl) verifySettingFieldsAreUnchanged(newPolicy *storage.Policy) error {
	return s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)

		oldPolicy, exists, err := s.getPolicy(newPolicy.GetId(), bucket)
		if err != nil {
			return err
		}

		if !exists {
			return nil
		}

		if oldPolicy.GetIsDefault() != newPolicy.GetIsDefault() {
			log.Warnf("'isDefault' is read-only fields. Setting it to previous value for policy %q.", newPolicy.GetName())
			newPolicy.IsDefault = oldPolicy.GetIsDefault()
		}

		if oldPolicy.GetCriteriaLocked() != newPolicy.GetCriteriaLocked() {
			log.Warnf("'criteriaLocked' is read-only fields. Setting it to previous value for policy %q.", newPolicy.GetName())
			newPolicy.CriteriaLocked = oldPolicy.GetCriteriaLocked()
		}

		var errs errorhelpers.ErrorList
		if oldPolicy.GetCriteriaLocked() {
			if !reflect.DeepEqual(oldPolicy.GetPolicySections(), newPolicy.GetPolicySections()) {
				errs.AddString("policy's criteria fields cannot be updated since they are marked read-only (criteriaLocked=true)")
			}
		}

		if oldPolicy.GetMitreVectorsLocked() != newPolicy.GetMitreVectorsLocked() {
			log.Warnf("'mitreVectorsLocked' is read-only field. Setting it to previous value for policy %q.", newPolicy.GetName())

			newPolicy.MitreVectorsLocked = oldPolicy.GetMitreVectorsLocked()
			newPolicy.CriteriaLocked = oldPolicy.GetCriteriaLocked()
		}

		if oldPolicy.GetMitreVectorsLocked() {
			if !reflect.DeepEqual(oldPolicy.GetMitreAttackVectors(), newPolicy.GetMitreAttackVectors()) {
				errs.AddString("policy's MITRE ATT&CK vectors cannot be updated since they are marked read-only (mitreVectorsLocked=true)")
			}
		}
		return errs.ToError()
	})
}
