package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	policyBucket = []byte("policies")

	log = logging.LoggerForModule()
)

// Store provides storage functionality for policies.
type Store interface {
	GetPolicy(id string) (*storage.Policy, bool, error)
	GetPolicies() ([]*storage.Policy, error)
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
	if policies, err := store.GetPolicies(); err != nil {
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
		// If the ID or Name already exists then ignore
		if policyIDSet.Contains(p.GetId()) || policyNameSet.Contains(p.GetName()) {
			continue
		}
		count++
		if _, err := store.AddPolicy(p); err != nil {
			panic(err)
		}
	}
	log.Infof("Loaded %d new default Policies", count)
}
