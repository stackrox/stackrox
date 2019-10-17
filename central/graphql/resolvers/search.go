package resolvers

import (
	"context"
	"math"

	searchService "github.com/stackrox/rox/central/search/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("searchOptions(categories: [SearchCategory!]): [String!]!"),
		schema.AddQuery("globalSearch(categories: [SearchCategory!], query: String!): [SearchResult!]!"),
		schema.AddQuery("searchAutocomplete(categories: [SearchCategory!], query: String!): [String!]!"),
		schema.AddInput("SortOption", []string{"field: String", "reversed: Boolean"}),
		schema.AddInput("Pagination", []string{"offset: Int", "limit: Int", "sortOption: SortOption"}),
	)
}

type sortOption struct {
	Field    *string
	Reversed *bool
}

func (s *sortOption) AsV1SortOption() *v1.SortOption {
	if s == nil {
		return nil
	}
	return &v1.SortOption{
		Field: func() string {
			if s.Field == nil {
				return ""
			}
			return *s.Field
		}(),
		Reversed: func() bool {
			if s.Reversed == nil {
				return false
			}
			return *s.Reversed
		}(),
	}
}

type pagination struct {
	Offset     *int32
	Limit      *int32
	SortOption *sortOption
}

func (r *pagination) AsV1Pagination() *v1.Pagination {
	if r == nil {
		return nil
	}
	return &v1.Pagination{
		Offset: func() int32 {
			if r.Offset == nil {
				return 0
			}
			return *r.Offset
		}(),
		Limit: func() int32 {
			if r.Limit == nil {
				return 0
			}
			return *r.Limit
		}(),
		SortOption: r.SortOption.AsV1SortOption(),
	}
}

type paginatedQuery struct {
	Query      *string
	Pagination *pagination
}

func (r *paginatedQuery) AsV1QueryOrEmpty() (*v1.Query, error) {
	var q *v1.Query
	if r == nil || r.Query == nil {
		q := search.EmptyQuery()
		paginated.FillPagination(q, r.Pagination.AsV1Pagination(), math.MaxInt32)
		return q, nil
	}
	q, err := search.ParseQuery(*r.Query, search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	paginated.FillPagination(q, r.Pagination.AsV1Pagination(), math.MaxInt32)
	return q, nil
}

func (r *paginatedQuery) String() string {
	if r == nil || r.Query == nil {
		return ""
	}
	return *r.Query
}

type rawQuery struct {
	Query *string
}

func (r rawQuery) AsV1QueryOrEmpty() (*v1.Query, error) {
	if r.Query == nil {
		return search.EmptyQuery(), nil
	}
	return search.ParseQuery(*r.Query, search.MatchAllIfEmpty())
}

func (r rawQuery) String() string {
	if r.Query == nil {
		return ""
	}
	return *r.Query
}

func (resolver *Resolver) getAutoCompleteSearchers() map[v1.SearchCategory]search.Searcher {
	searchers := map[v1.SearchCategory]search.Searcher{
		v1.SearchCategory_ALERTS:           resolver.ViolationsDataStore,
		v1.SearchCategory_CLUSTERS:         resolver.ClusterDataStore,
		v1.SearchCategory_DEPLOYMENTS:      resolver.DeploymentDataStore,
		v1.SearchCategory_IMAGES:           resolver.ImageDataStore,
		v1.SearchCategory_POLICIES:         resolver.PolicyDataStore,
		v1.SearchCategory_SECRETS:          resolver.SecretsDataStore,
		v1.SearchCategory_NAMESPACES:       resolver.NamespaceDataStore,
		v1.SearchCategory_NODES:            resolver.NodeGlobalDataStore,
		v1.SearchCategory_COMPLIANCE:       resolver.ComplianceAggregator,
		v1.SearchCategory_SERVICE_ACCOUNTS: resolver.ServiceAccountsDataStore,
		v1.SearchCategory_ROLES:            resolver.K8sRoleStore,
		v1.SearchCategory_ROLEBINDINGS:     resolver.K8sRoleBindingStore,
	}

	return searchers
}

func (resolver *Resolver) getSearchFuncs() map[v1.SearchCategory]searchService.SearchFunc {

	searchfuncs := map[v1.SearchCategory]searchService.SearchFunc{
		v1.SearchCategory_ALERTS:           resolver.ViolationsDataStore.SearchAlerts,
		v1.SearchCategory_CLUSTERS:         resolver.ClusterDataStore.SearchResults,
		v1.SearchCategory_DEPLOYMENTS:      resolver.DeploymentDataStore.SearchDeployments,
		v1.SearchCategory_IMAGES:           resolver.ImageDataStore.SearchImages,
		v1.SearchCategory_POLICIES:         resolver.PolicyDataStore.SearchPolicies,
		v1.SearchCategory_SECRETS:          resolver.SecretsDataStore.SearchSecrets,
		v1.SearchCategory_NAMESPACES:       resolver.NamespaceDataStore.SearchResults,
		v1.SearchCategory_NODES:            resolver.NodeGlobalDataStore.SearchResults,
		v1.SearchCategory_SERVICE_ACCOUNTS: resolver.ServiceAccountsDataStore.SearchServiceAccounts,
		v1.SearchCategory_ROLES:            resolver.K8sRoleStore.SearchRoles,
		v1.SearchCategory_ROLEBINDINGS:     resolver.K8sRoleBindingStore.SearchRoleBindings,
	}

	return searchfuncs
}

func checkSearchAuth(ctx context.Context) error {
	for _, resourceMetadata := range searchService.GetSearchCategoryToResourceMetadata() {
		if err := readAuth(resourceMetadata)(ctx); err != nil {
			return err
		}
	}
	return nil
}

type searchRequest struct {
	Query      string
	Categories *[]string
}

// SearchAutocomplete returns autocomplete responses for the given partial query.
func (resolver *Resolver) SearchAutocomplete(ctx context.Context, args searchRequest) ([]string, error) {
	if err := checkSearchAuth(ctx); err != nil {
		return nil, err
	}
	return searchService.RunAutoComplete(ctx, args.Query, toSearchCategories(args.Categories), resolver.getAutoCompleteSearchers())
}

// SearchOptions gets all search options available for the listed categories
func (resolver *Resolver) SearchOptions(ctx context.Context, args struct{ Categories *[]string }) ([]string, error) {
	if err := checkSearchAuth(ctx); err != nil {
		return nil, err
	}
	return searchService.Options(toSearchCategories(args.Categories)), nil
}

// GlobalSearch returns search results for the given categories and query.
// Note: there is not currently a way to request the underlying object from SearchResult; it might be nice to have
// this in the future.
func (resolver *Resolver) GlobalSearch(ctx context.Context, args searchRequest) ([]*searchResultResolver, error) {
	if err := checkSearchAuth(ctx); err != nil {
		return nil, err
	}
	results, _, err := searchService.GlobalSearch(ctx, args.Query, toSearchCategories(args.Categories), resolver.getSearchFuncs())
	return resolver.wrapSearchResults(results, err)
}
