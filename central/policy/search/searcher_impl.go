package search

import (
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

// SearchRawPolicies retrieves Policies from the indexer and storage
func (ds *searcherImpl) SearchRawPolicies(q *v1.Query) ([]*storage.Policy, error) {
	policies, _, err := ds.searchPolicies(q)
	return policies, err
}

// SearchPolicies retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchPolicies(q *v1.Query) ([]*v1.SearchResult, error) {
	policies, results, err := ds.searchPolicies(q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(policies))
	for i, policy := range policies {
		protoResults = append(protoResults, convertPolicy(policy, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchPolicies(q *v1.Query) ([]*storage.Policy, []search.Result, error) {
	results, err := ds.indexer.SearchPolicies(q)
	if err != nil {
		return nil, nil, err
	}
	var policies []*storage.Policy
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
func convertPolicy(policy *storage.Policy, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_POLICIES,
		Id:             policy.GetId(),
		Name:           policy.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
