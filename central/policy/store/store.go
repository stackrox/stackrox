package store

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	bolt "go.etcd.io/bbolt"
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
	UpsertPolicy(policy *storage.Policy) error
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
	return s
}
