package store

import (
	"fmt"
	"sort"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type storeImpl struct {
	*bolt.DB

	mutex sync.Mutex
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

// GetAllPolicies retrieves policies matching the request from bolt
func (b *storeImpl) GetAllPolicies() ([]*storage.Policy, error) {
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

func (b *storeImpl) GetPolicies(ids ...string) ([]*storage.Policy, []int, []error, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Policy")
	var policies []*storage.Policy
	var missingIndices []int
	var errorList []error
	err := b.View(func(tx *bolt.Tx) error {
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
func (b *storeImpl) AddPolicy(policy *storage.Policy) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Policy")

	if policy.Id == "" {
		policy.Id = uuid.NewV4().String()
	}

	// Lock here so we can check whether the policy exists outside of the Bolt write lock.  This can race with
	// update/delete but I don't think this results in any problems.
	b.mutex.Lock()
	defer b.mutex.Unlock()

	equalPolicyExists := false
	err := b.View(func(tx *bolt.Tx) error {
		var errs []error
		bucket := tx.Bucket(policyBucket)

		// Check whether a policy with this ID already exists
		existingPolicy, exists, err := b.getPolicy(policy.GetId(), bucket)
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

	err = b.Update(func(tx *bolt.Tx) error {
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
	})
	return policy.Id, err
}

// UpdatePolicy updates a policy to bolt
func (b *storeImpl) UpdatePolicy(policy *storage.Policy) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Policy")

	// Have to lock here because this is an upsert, not an update.  AddPolicy should not re-create a policy which is
	// created here.
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, policyBucket, policy.GetId()); val != policy.GetName() {
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
