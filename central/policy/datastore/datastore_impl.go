package datastore

import (
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchPolicies
func (ds *datastoreImpl) SearchPolicies(q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchPolicies(q)
}

// SearchRawPolicies
func (ds *datastoreImpl) SearchRawPolicies(q *v1.Query) ([]*v1.Policy, error) {
	return ds.searcher.SearchRawPolicies(q)
}

func (ds *datastoreImpl) GetPolicy(id string) (*v1.Policy, bool, error) {
	return ds.storage.GetPolicy(id)
}

func (ds *datastoreImpl) GetPolicies() ([]*v1.Policy, error) {
	return ds.storage.GetPolicies()
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *datastoreImpl) AddPolicy(policy *v1.Policy) (string, error) {
	id, err := ds.storage.AddPolicy(policy)
	if err != nil {
		return id, err
	}
	return id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *datastoreImpl) UpdatePolicy(policy *v1.Policy) error {
	if err := ds.storage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *datastoreImpl) RemovePolicy(id string) error {
	if err := ds.storage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

func (ds *datastoreImpl) RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error {
	return ds.storage.RenamePolicyCategory(request)
}

func (ds *datastoreImpl) DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error {
	return ds.storage.DeletePolicyCategory(request)
}
