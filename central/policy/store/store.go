package store

import (
	policyUtils "github.com/stackrox/stackrox/central/policy/utils"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/defaults/policies"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

var (
	policyBucket = []byte("policies")
	// As of 66.0, any default policies that were added from 65.0 onwards cannot be deleted.
	// Any policies added prior to 65.0 can be deleted because the criteria fields are not locked.
	// Locked policy criteria guarantees that the criteria remains unchanged as is as it was shipped.
	removedDefaultPolicyBucket = []byte("removed_default_policies")

	log = logging.LoggerForModule()
)

// PolicyStoreErrorList is used to encapsulate multiple errors returned from policy store methods
type PolicyStoreErrorList struct {
	Errors []error
}

func (p *PolicyStoreErrorList) Error() string {
	return errorhelpers.NewErrorListWithErrors("policy store encountered errors", p.Errors).String()
}

// IDConflictError can be returned by AddPolicies when a policy exists with the same ID as a new policy
type IDConflictError struct {
	ErrString          string
	ExistingPolicyName string
}

func (i *IDConflictError) Error() string {
	return i.ErrString
}

// NameConflictError can be returned by AddPolicies when a policy exists with the same name as a new policy
type NameConflictError struct {
	ErrString          string
	ExistingPolicyName string
}

func (i *NameConflictError) Error() string {
	return i.ErrString
}

// Store provides storage functionality for policies.
//go:generate mockgen-wrapper
type Store interface {
	GetPolicy(id string) (*storage.Policy, bool, error)
	GetAllPolicies() ([]*storage.Policy, error)
	GetPolicies(ids ...string) ([]*storage.Policy, []int, []error, error)
	AddPolicy(policy *storage.Policy, removePolicyTombstone bool) (string, error)
	UpdatePolicy(*storage.Policy) error
	RemovePolicy(id string) error
	RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error
	DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	bolthelper.RegisterBucketOrPanic(db, removedDefaultPolicyBucket)
	s := &storeImpl{
		DB: db,
	}
	s.addDefaults()
	return s
}

// New returns a new Store instance using the provided bolt DB instance.
func newWithoutDefaults(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	bolthelper.RegisterBucketOrPanic(db, removedDefaultPolicyBucket)
	return &storeImpl{
		DB: db,
	}
}

func (s *storeImpl) addDefaults() {
	policyIDSet, policyNameSet := set.NewStringSet(), set.NewStringSet()
	storedPolicies, err := s.GetAllPolicies()
	if err != nil {
		panic(err)
	}

	for _, p := range storedPolicies {
		policyIDSet.Add(p.GetId())
		policyNameSet.Add(p.GetName())
	}

	// Preload the default policies.
	defaultPolicies, err := policies.DefaultPolicies()
	// Hard panic here is okay, since we can always guarantee that we will be able to get the default policies out.
	utils.CrashOnError(err)

	var count int
	for _, p := range defaultPolicies {
		wasRemoved, err := s.wasDefaultPolicyRemoved(p.GetId())
		if err != nil {
			// Log error and continue adding the policy. We will panic during adding the policy, if the DB error is persistent.
			log.Errorf("Could not determine if the default policy %s was previously removed. Continuing with adding it to DB: %v", p.GetId(), err)
		}
		// In case of error, wasDefaultPolicyRemoved is true to not block policy addition.
		if wasRemoved {
			continue
		}

		// If the ID or Name already exists then ignore
		if policyIDSet.Contains(p.GetId()) {
			if err := s.updatePolicyTombstone(p.GetId(), false, true); err != nil {
				panic(err)
			}
			continue
		}

		// If ID is not the same as the shipped default policy, we treat it as custom policy. Hence, the tombstone
		// state is not tracked.
		if policyNameSet.Contains(p.GetName()) {
			continue
		}
		count++

		// fill multi-word sort helper field
		policyUtils.FillSortHelperFields(p)

		if _, err := s.AddPolicy(p, false); err != nil {
			panic(err)
		}

		if err = s.updatePolicyTombstone(p.GetId(), false, true); err != nil {
			panic(err)
		}
	}
	log.Infof("Loaded %d new default Policies", count)
}
