package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

var networkPolicyBucket = []byte("networkpolicies")

var (
	log = logging.LoggerForModule()
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper Store
type Store interface {
	GetNetworkPolicy(id string) (*storage.NetworkPolicy, bool, error)
	GetNetworkPolicies(clusterID, namespace string) ([]*storage.NetworkPolicy, error)
	CountMatchingNetworkPolicies(clusterID, namespace string) (int, error)
	AddNetworkPolicy(np *storage.NetworkPolicy) error
	UpdateNetworkPolicy(np *storage.NetworkPolicy) error
	RemoveNetworkPolicy(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, networkPolicyBucket)
	return &storeImpl{
		DB: db,
	}
}
