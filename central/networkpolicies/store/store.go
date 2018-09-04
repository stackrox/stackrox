package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

const networkPolicyBucket = "networkpolicies"

var (
	log = logging.LoggerForModule()
)

// Store provides storage functionality for alerts.
//go:generate mockery -name=Store
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
