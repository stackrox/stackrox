package store

import (
	"sort"
	"strings"

	bolt "github.com/etcd-io/bbolt"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	policyBucket = []byte("policies")

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
	AddPolicy(*storage.Policy) (string, error)
	UpdatePolicy(*storage.Policy) error
	RemovePolicy(id string) error
	RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error
	DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	s := &storeImpl{
		DB: db,
	}
	addDefaults(s)
	return s
}

// New returns a new Store instance using the provided bolt DB instance.
func newWithoutDefaults(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	return &storeImpl{
		DB: db,
	}
}

func addDefaults(store Store) {
	policyIDSet := set.NewStringSet()
	policyNameSet := set.NewStringSet()
	if policies, err := store.GetAllPolicies(); err != nil {
		panic(err)
	} else if len(policies) > 0 {
		for _, p := range policies {
			policyIDSet.Add(p.GetId())
			policyNameSet.Add(p.GetName())
		}
	}

	// Preload the default policies.
	policies, err := defaults.Policies()
	if err != nil {
		panic(err)
	}
	var count int
	for _, p := range policies {
		if !features.BooleanPolicyLogic.Enabled() && booleanpolicy.IsBooleanPolicy(p) {
			continue
		}
		if features.BooleanPolicyLogic.Enabled() {
			// Hard panic here is okay, since this is a default policy, and we can guarantee that
			// all default policies can be converted.
			utils.Must(booleanpolicy.EnsureConverted(p))
		}
		// If the ID or Name already exists then ignore
		if policyIDSet.Contains(p.GetId()) || policyNameSet.Contains(p.GetName()) {
			continue
		}
		count++

		// fill multi-word sort helper field
		FillSortHelperFields(p)

		if _, err := store.AddPolicy(p); err != nil {
			panic(err)
		}
	}
	log.Infof("Loaded %d new default Policies", count)
}

// FillSortHelperFields fills multi word sort fields such as Name, Lifecycle Stages etc.
func FillSortHelperFields(policies ...*storage.Policy) {
	for _, policy := range policies {
		policy.SORTName = policy.Name

		sort.Slice(policy.GetLifecycleStages(), func(i, j int) bool {
			return policy.GetLifecycleStages()[i].String() < policy.GetLifecycleStages()[j].String()
		})
		var stages []string
		for _, lifecycleStage := range policy.GetLifecycleStages() {
			stages = append(stages, lifecycleStage.String())
		}
		policy.SORTLifecycleStage = strings.Join(stages, ",")

		if len(policy.GetEnforcementActions()) > 0 {
			policy.SORTEnforcement = true
		}
	}
}
