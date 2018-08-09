package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const policyBucket = "policies"

// Store provides storage functionality for alerts.
type Store interface {
	GetPolicy(id string) (*v1.Policy, bool, error)
	GetPolicies() ([]*v1.Policy, error)
	AddPolicy(*v1.Policy) (string, error)
	UpdatePolicy(*v1.Policy) error
	RemovePolicy(id string) error
	RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error
	DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	return &storeImpl{
		DB: db,
	}
}
