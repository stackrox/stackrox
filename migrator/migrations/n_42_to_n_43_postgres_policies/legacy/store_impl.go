package store

import (
	"context"
	"encoding/json"
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
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/sync"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB

	mutex sync.Mutex
}

func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Policy")
	var policyIDs []string
	err := s.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(policyBucket)
		return b.ForEach(func(k, v []byte) error {
			policyIDs = append(policyIDs, string(k))
			return nil
		})
	})
	return policyIDs, err
}

func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storage.Policy) error {
	return errors.New("not implemented")
}

func (s *storeImpl) DeleteMany(ctx context.Context, ids []string) error {
	return errors.New("not implemented")
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
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

// Get returns policy with given id.
func (s *storeImpl) Get(_ context.Context, id string) (policy *storage.Policy, exists bool, err error) {
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

// GetAll retrieves policies matching the request from bolt
func (s *storeImpl) GetAll(_ context.Context) ([]*storage.Policy, error) {
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

func (s *storeImpl) GetMany(_ context.Context, ids ...string) ([]*storage.Policy, []int, []error, error) {
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

// Upsert updates a policy to bolt
func (s *storeImpl) Upsert(_ context.Context, policy *storage.Policy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Policy")

	// Have to lock here because this is an upsert, not an update.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	wasRemoved, err := s.wasDefaultPolicyRemoved(policy.GetId())
	if err != nil {
		return err
	}
	if wasRemoved {
		return errors.Errorf("default policy %s was previously removed", policy.GetId())
	}
	equalPolicyExists := false
	err = s.View(func(tx *bolt.Tx) error {
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
		}
		return nil
	})

	if err != nil {
		return err
	}
	if equalPolicyExists {
		return nil
	}

	// Verify that policy's settings flags are not updated. If the lock flags are set to read-only,
	// verify that the corresponding fields are not updated.
	if err := s.verifySettingFieldsAreUnchanged(policy); err != nil {
		return err
	}

	err = s.Update(func(tx *bolt.Tx) error {
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

// Delete removes a policy.
func (s *storeImpl) Delete(_ context.Context, id string) error {
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

	return nil
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
			log.Warnf("'isDefault' is a read-only field. Setting it to previous value for policy %q.", newPolicy.GetName())
			newPolicy.IsDefault = oldPolicy.GetIsDefault()
		}

		if oldPolicy.GetCriteriaLocked() != newPolicy.GetCriteriaLocked() {
			log.Warnf("'criteriaLocked' is a read-only field. Setting it to previous value for policy %q.", newPolicy.GetName())
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

func (b *storeImpl) Walk(_ context.Context, fn func(np *storage.Policy) error) error {
	return b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var np storage.Policy
			if err := proto.Unmarshal(v, &np); err != nil {
				return err
			}
			return fn(&np)
		})
	})
}
