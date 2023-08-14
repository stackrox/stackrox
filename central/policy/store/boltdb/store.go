package boltdb

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/policyutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

var (
	policyBucket = []byte("policies")
	// As of 66.0, any default policies that were added from 65.0 onwards cannot be deleted.
	// Any policies added prior to 65.0 can be deleted because the criteria fields are not locked.
	// Locked policy criteria guarantees that the criteria remains unchanged as is as it was shipped.
	removedDefaultPolicyBucket = []byte("removed_default_policies")

	policyCtx = context.Background()

	log = logging.LoggerForModule()
)

// Store provides storage functionality for policies.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.Policy, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Policy, []int, error)

	GetAll(ctx context.Context) ([]*storage.Policy, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, policy *storage.Policy) error
	UpsertMany(ctx context.Context, objs []*storage.Policy) error

	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
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
	policyIDSet := set.NewStringSet()
	storedPolicies, err := s.GetAll(policyCtx)
	if err != nil {
		panic(err)
	}

	for _, p := range storedPolicies {
		policyIDSet.Add(p.GetId())
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

		// If ID is not the same as the shipped default policy, we treat it as custom policy. Hence, the tombstone
		// state is not tracked.
		if policyIDSet.Contains(p.GetId()) {
			continue
		}
		count++

		// fill multi-word sort helper field
		policyutils.FillSortHelperFields(p)

		if err := s.Upsert(policyCtx, p); err != nil {
			panic(err)
		}
	}
	log.Infof("Loaded %d new default Policies", count)
}
