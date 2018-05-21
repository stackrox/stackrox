package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/defaults"
)

// PolicyDataStore provides an intermediary implementation layer for PolicyStorage.
type PolicyDataStore interface {
	db.PolicyStorage

	SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, error)
}

// NewPolicyDataStore provides a new instance of PolicyDataStore
func NewPolicyDataStore(storage db.PolicyStorage, indexer search.PolicyIndex) (PolicyDataStore, error) {
	ds := &policyDataStoreImpl{
		PolicyStorage: storage,
		indexer:       indexer,
	}
	if err := ds.loadDefaults(); err != nil {
		return nil, err
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

// PolicyDataStore provides an intermediary implementation layer for PolicyStorage.
type policyDataStoreImpl struct {
	// This is an embedded type so we don't have to override all functions. Indexing is a subset of Storage
	db.PolicyStorage

	indexer search.PolicyIndex
}

func (ds *policyDataStoreImpl) buildIndex() error {
	policies, err := ds.GetPolicies()
	if err != nil {
		return err
	}
	for _, p := range policies {
		if err := ds.indexer.AddPolicy(p); err != nil {
			logger.Errorf("Error inserting policy %s (%s) into index: %s", p.GetId(), p.GetName(), err)
		}
	}
	return nil
}

func (ds *policyDataStoreImpl) loadDefaults() error {
	if policies, err := ds.GetPolicies(); err == nil && len(policies) > 0 {
		return nil
	}

	policies, err := defaults.Policies()
	if err != nil {
		return err
	}

	for _, p := range policies {
		if _, err := ds.PolicyStorage.AddPolicy(p); err != nil {
			return err
		}
	}

	logger.Infof("Loaded %d default Policies", len(policies))
	return nil
}

// SearchRawPolicies retrieves Policies from the indexer and storage
func (ds *policyDataStoreImpl) SearchRawPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, error) {
	policies, _, err := ds.searchPolicies(request)
	return policies, err
}

// SearchPolicies retrieves SearchResults from the indexer and storage
func (ds *policyDataStoreImpl) SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	policies, results, err := ds.searchPolicies(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(policies))
	for i, policy := range policies {
		protoResults = append(protoResults, search.ConvertPolicy(policy, results[i]))
	}
	return protoResults, nil
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *policyDataStoreImpl) AddPolicy(policy *v1.Policy) (string, error) {
	id, err := ds.PolicyStorage.AddPolicy(policy)
	if err != nil {
		return id, err
	}
	return id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *policyDataStoreImpl) UpdatePolicy(policy *v1.Policy) error {
	if err := ds.PolicyStorage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *policyDataStoreImpl) RemovePolicy(id string) error {
	if err := ds.PolicyStorage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

func (ds *policyDataStoreImpl) searchPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, []search.Result, error) {
	results, err := ds.indexer.SearchPolicies(request)
	if err != nil {
		return nil, nil, err
	}
	var policies []*v1.Policy
	var newResults []search.Result
	for _, result := range results {
		policy, exists, err := ds.GetPolicy(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		policies = append(policies, policy)
		newResults = append(newResults, result)
	}
	return policies, newResults, nil
}
