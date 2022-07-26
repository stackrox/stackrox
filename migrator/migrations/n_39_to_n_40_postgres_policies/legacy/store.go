package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	bolt "go.etcd.io/bbolt"
)

var (
	policyBucket = []byte("policies")
	// As of 66.0, any default policies that were added from 65.0 onwards cannot be deleted.
	// Any policies added prior to 65.0 can be deleted because the criteria fields are not locked.
	// Locked policy criteria guarantees that the criteria remains unchanged as is as it was shipped.
	removedDefaultPolicyBucket = []byte("removed_default_policies")

	policyCtx = sac.WithAllAccess(context.Background())

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
type Store interface {
	Get(ctx context.Context, id string) (*storage.Policy, bool, error)
	GetMany(ctx context.Context, ids ...string) ([]*storage.Policy, []int, []error, error)
	Walk(ctx context.Context, fn func(np *storage.Policy) error) error

	GetAll(ctx context.Context) ([]*storage.Policy, error)
	GetIDs(ctx context.Context) ([]string, error)

	Upsert(ctx context.Context, policy *storage.Policy) error
	UpsertMany(ctx context.Context, objs []*storage.Policy) error

	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, policyBucket)
	bolthelper.RegisterBucketOrPanic(db, removedDefaultPolicyBucket)
	s := &storeImpl{
		DB: db,
	}
	return s
}
