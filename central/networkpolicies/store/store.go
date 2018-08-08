package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/boltdb/bolt"
)

const networkPolicyBucket = "networkpolicies"

var (
	log = logging.LoggerForModule()
)

// Store provides storage functionality for alerts.
type Store interface {
	GetNetworkPolicy(id string) (*v1.NetworkPolicy, bool, error)
	GetNetworkPolicies(request *v1.GetNetworkPoliciesRequest) ([]*v1.NetworkPolicy, error)
	CountNetworkPolicies() (int, error)
	AddNetworkPolicy(np *v1.NetworkPolicy) error
	UpdateNetworkPolicy(np *v1.NetworkPolicy) error
	RemoveNetworkPolicy(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, networkPolicyBucket)
	return &storeImpl{
		DB: db,
	}
}
