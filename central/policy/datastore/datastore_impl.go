package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	policySAC = sac.ForResource(resources.Policy)
)

type datastoreImpl struct {
	storage    store.Store
	indexer    index.Indexer
	searcher   search.Searcher
	keyedMutex *concurrency.KeyedMutex
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return ds.indexer.Search(q)
}

// SearchPolicies
func (ds *datastoreImpl) SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchPolicies(ctx, q)
}

// SearchRawPolicies
func (ds *datastoreImpl) SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error) {
	return ds.searcher.SearchRawPolicies(ctx, q)
}

func (ds *datastoreImpl) GetPolicy(ctx context.Context, id string) (*storage.Policy, bool, error) {
	policy, exists, err := ds.storage.GetPolicy(id)
	if err != nil || !exists {
		return nil, false, err
	}

	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return policy, true, nil
}

func (ds *datastoreImpl) GetPolicies(ctx context.Context) ([]*storage.Policy, error) {
	policies, err := ds.storage.GetPolicies()
	if err != nil {
		return nil, err
	}

	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	return policies, err
}

// GetPolicyByName returns policy with given name.
func (ds *datastoreImpl) GetPolicyByName(ctx context.Context, name string) (policy *storage.Policy, exists bool, err error) {
	policies, err := ds.GetPolicies(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, p := range policies {
		if p.GetName() == name {
			if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
				return nil, false, err
			}
			return p, true, nil
		}
	}
	return nil, false, nil
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *datastoreImpl) AddPolicy(ctx context.Context, policy *storage.Policy) (string, error) {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", errors.New("permission denied")
	}

	// No need to lock here because nobody can update the policy
	// until this function returns and they receive the id.
	id, err := ds.storage.AddPolicy(policy)
	if err != nil {
		return id, err
	}
	return id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *datastoreImpl) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.keyedMutex.Lock(policy.GetId())
	defer ds.keyedMutex.Unlock(policy.GetId())
	if err := ds.storage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *datastoreImpl) RemovePolicy(ctx context.Context, id string) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)
	if err := ds.storage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

func (ds *datastoreImpl) RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.RenamePolicyCategory(request)
}

func (ds *datastoreImpl) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.DeletePolicyCategory(request)
}
