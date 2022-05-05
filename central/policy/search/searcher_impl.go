package search

import (
	"context"

	"github.com/stackrox/rox/central/policy/index"
	policyMapping "github.com/stackrox/rox/central/policy/index/mappings"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.SORTPolicyName.String(),
	}

	policySAC = sac.ForResource(resources.Policy)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchRawPolicies retrieves Policies from the indexer and storage
func (ds *searcherImpl) SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error) {
	policies, _, err := ds.searchPolicies(ctx, q)
	return policies, err
}

// Search retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	policies, results, err := ds.searchPolicies(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(policies))
	for i, policy := range policies {
		protoResults = append(protoResults, convertPolicy(policy, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	var policies []*storage.Policy
	var newResults []search.Result
	for _, result := range results {
		policy, exists, err := ds.storage.Get(ctx, result.ID)
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

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	return ds.searcher.Search(ctx, q)
}

func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return 0, err
	}

	return ds.searcher.Count(ctx, q)
}

// convertPolicy returns proto search result from a policy object and the internal search result
func convertPolicy(policy *storage.Policy, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_POLICIES,
		Id:             policy.GetId(),
		Name:           policy.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	safeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(unsafeSearcher)
	transformedSortFieldSearcher := sortfields.TransformSortFields(safeSearcher, policyMapping.OptionsMap)
	paginatedSearcher := paginated.Paginated(transformedSortFieldSearcher)
	return paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
}
