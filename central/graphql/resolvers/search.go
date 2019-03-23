package resolvers

import (
	"context"

	searchService "github.com/stackrox/rox/central/search/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("searchOptions(categories: [SearchCategory!]): [String!]!"),
		schema.AddQuery("globalSearch(categories: [SearchCategory!], query: String!): [SearchResult!]!"),
		schema.AddQuery("searchAutocomplete(categories: [SearchCategory!], query: String!): [String!]!"),
	)
}

type rawQuery struct {
	Query *string
}

func (r rawQuery) AsV1Query() (*v1.Query, error) {
	if r.Query == nil {
		return nil, nil
	}
	return search.ParseRawQuery(*r.Query)
}

func (resolver *Resolver) getAutoCompleteSearchers() map[v1.SearchCategory]search.Searcher {
	searchers := map[v1.SearchCategory]search.Searcher{
		v1.SearchCategory_ALERTS:      resolver.ViolationsDataStore,
		v1.SearchCategory_DEPLOYMENTS: resolver.DeploymentDataStore,
		v1.SearchCategory_IMAGES:      resolver.ImageDataStore,
		v1.SearchCategory_POLICIES:    resolver.PolicyDataStore,
		v1.SearchCategory_SECRETS:     resolver.SecretsDataStore,
		v1.SearchCategory_NAMESPACES:  resolver.NamespaceDataStore,
		v1.SearchCategory_NODES:       resolver.NodeGlobalStore,
		v1.SearchCategory_COMPLIANCE:  resolver.ComplianceAggregator,
	}

	if features.K8sRBAC.Enabled() {
		searchers[v1.SearchCategory_SERVICE_ACCOUNTS] = resolver.ServiceAccountsDataStore
	}

	return searchers
}

func (resolver *Resolver) getSearchFuncs() map[v1.SearchCategory]searchService.SearchFunc {

	searchfuncs := map[v1.SearchCategory]searchService.SearchFunc{
		v1.SearchCategory_ALERTS:      resolver.ViolationsDataStore.SearchAlerts,
		v1.SearchCategory_DEPLOYMENTS: resolver.DeploymentDataStore.SearchDeployments,
		v1.SearchCategory_IMAGES:      resolver.ImageDataStore.SearchImages,
		v1.SearchCategory_POLICIES:    resolver.PolicyDataStore.SearchPolicies,
		v1.SearchCategory_SECRETS:     resolver.SecretsDataStore.SearchSecrets,
		v1.SearchCategory_NAMESPACES:  resolver.NamespaceDataStore.SearchResults,
		v1.SearchCategory_NODES:       resolver.NodeGlobalStore.SearchResults,
	}

	if features.K8sRBAC.Enabled() {
		searchfuncs[v1.SearchCategory_SERVICE_ACCOUNTS] = resolver.ServiceAccountsDataStore.SearchServiceAccounts
	}

	return searchfuncs
}

func checkSearchAuth(ctx context.Context) error {
	for _, resource := range searchService.GetSearchCategoryToResource() {
		if err := readAuth(resource)(ctx); err != nil {
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
	return searchService.RunAutoComplete(args.Query, toSearchCategories(args.Categories), resolver.getAutoCompleteSearchers())
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
	results, _, err := searchService.GlobalSearch(args.Query, toSearchCategories(args.Categories), resolver.getSearchFuncs())
	return resolver.wrapSearchResults(results, err)
}
