package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var policyLoaderType = reflect.TypeOf(storage.Policy{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.Policy{}), func() interface{} {
		return NewPolicyLoader(policyDataStore.Singleton())
	})
}

// NewPolicyLoader creates a new loader for policy data.
func NewPolicyLoader(ds policyDataStore.DataStore) PolicyLoader {
	return &policyLoaderImpl{
		loaded:   make(map[string]*storage.Policy),
		policyDS: ds,
	}
}

// GetPolicyLoader returns the PolicyLoader from the context if it exists.
func GetPolicyLoader(ctx context.Context) (PolicyLoader, error) {
	loader, err := GetLoader(ctx, policyLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(PolicyLoader), nil
}

// PolicyLoader loads policy data for other ops in the same context to use.
type PolicyLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.Policy, error)
	FromID(ctx context.Context, id string) (*storage.Policy, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Policy, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// policyLoaderImpl implements the PolicyDataLoader interface.
type policyLoaderImpl struct {
	lock     sync.RWMutex
	loaded   map[string]*storage.Policy
	policyDS policyDataStore.DataStore
}

// FromIDs loads a set of policies from a set of ids.
func (idl *policyLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.Policy, error) {
	policies, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return policies, nil
}

// FromID loads an policy from an ID.
func (idl *policyLoaderImpl) FromID(ctx context.Context, id string) (*storage.Policy, error) {
	policies, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return policies[0], nil
}

// FromQuery loads a set of policies that match a query.
func (idl *policyLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Policy, error) {
	results, err := idl.policyDS.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

// CountFromQuery returns the number of policies that match a given query.
func (idl *policyLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	numResults, err := idl.policyDS.Count(ctx, query)
	if err != nil {
		return 0, err
	}

	return int32(numResults), nil
}

// CountFromQuery returns the total number of policies.
func (idl *policyLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.CountFromQuery(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *policyLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.Policy, error) {
	policies, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		policies, err = idl.policyDS.SearchRawPolicies(ctx,
			search.NewQueryBuilder().AddDocIDs(collectMissing(ids, missing)...).ProtoQuery())
		if err != nil {
			return nil, err
		}

		idl.setAll(policies)
		policies, missing = idl.readAll(ids)
	}

	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all policies could be found: %s", strings.Join(missingIDs, ","))
	}

	return policies, nil
}

func (idl *policyLoaderImpl) setAll(policies []*storage.Policy) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, policy := range policies {
		idl.loaded[policy.GetId()] = policy
	}
}

func (idl *policyLoaderImpl) readAll(ids []string) (policies []*storage.Policy, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		policy, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			policies = append(policies, policy)
		}
	}
	return
}
