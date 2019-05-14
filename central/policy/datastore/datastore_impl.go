package datastore

import (
	"context"

	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage    store.Store
	indexer    index.Indexer
	searcher   search.Searcher
	keyedMutex *concurrency.KeyedMutex
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.indexer.Search(q)
}

// SearchPolicies
func (ds *datastoreImpl) SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchPolicies(q)
}

// SearchRawPolicies
func (ds *datastoreImpl) SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error) {
	return ds.searcher.SearchRawPolicies(q)
}

func (ds *datastoreImpl) GetPolicy(ctx context.Context, id string) (*storage.Policy, bool, error) {
	return ds.storage.GetPolicy(id)
}

func (ds *datastoreImpl) GetPolicies(ctx context.Context) ([]*storage.Policy, error) {
	return ds.storage.GetPolicies()
}

// GetPolicyByName returns policy with given name.
func (ds *datastoreImpl) GetPolicyByName(ctx context.Context, name string) (policy *storage.Policy, exists bool, err error) {
	policies, err := ds.GetPolicies(ctx)
	if err != nil {
		return nil, false, err
	}
	for _, p := range policies {
		if p.GetName() == name {
			return p, true, nil
		}
	}
	return nil, false, nil
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *datastoreImpl) AddPolicy(ctx context.Context, policy *storage.Policy) (string, error) {
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
	ds.keyedMutex.Lock(policy.GetId())
	defer ds.keyedMutex.Unlock(policy.GetId())
	if err := ds.storage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *datastoreImpl) RemovePolicy(ctx context.Context, id string) error {
	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)
	if err := ds.storage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

func (ds *datastoreImpl) RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) error {
	return ds.storage.RenamePolicyCategory(request)
}

func (ds *datastoreImpl) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) error {
	return ds.storage.DeletePolicyCategory(request)
}
