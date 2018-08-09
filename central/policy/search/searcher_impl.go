package search

import (
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/search"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (ds *searcherImpl) buildIndex() error {
	policies, err := ds.storage.GetPolicies()
	if err != nil {
		return err
	}
	return ds.indexer.AddPolicies(policies)
}

func (ds *searcherImpl) loadDefaults() error {
	if policies, err := ds.storage.GetPolicies(); err == nil && len(policies) > 0 {
		return nil
	}

	policies, err := defaults.Policies()
	if err != nil {
		return err
	}

	for _, p := range policies {
		if _, err := ds.storage.AddPolicy(p); err != nil {
			return err
		}
	}

	log.Infof("Loaded %d default Policies", len(policies))
	return nil
}

// SearchRawPolicies retrieves Policies from the indexer and storage
func (ds *searcherImpl) SearchRawPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, error) {
	policies, _, err := ds.searchPolicies(request)
	return policies, err
}

// SearchPolicies retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	policies, results, err := ds.searchPolicies(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(policies))
	for i, policy := range policies {
		protoResults = append(protoResults, convertPolicy(policy, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, []search.Result, error) {
	results, err := ds.indexer.SearchPolicies(request)
	if err != nil {
		return nil, nil, err
	}
	var policies []*v1.Policy
	var newResults []search.Result
	for _, result := range results {
		policy, exists, err := ds.storage.GetPolicy(result.ID)
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

// ConvertPolicy returns proto search result from a policy object and the internal search result
func convertPolicy(policy *v1.Policy, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_POLICIES,
		Id:             policy.GetId(),
		Name:           policy.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
