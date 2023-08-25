package resolvers

import (
	"context"
	"math"

	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/rbac/service"
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
	)
}

// PaginatedQuery represents a query with pagination info
type PaginatedQuery struct {
	Query      *string
	ScopeQuery *string
	Pagination *inputtypes.Pagination
}

// PaginationWrapper represents pagination without any query
type PaginationWrapper struct {
	Pagination *inputtypes.Pagination
}

// AsV1QueryOrEmpty returns a proto query or empty proto query if pagination query is empty
func (r *PaginatedQuery) AsV1QueryOrEmpty() (*v1.Query, error) {
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

// AsV1ScopeQueryOrEmpty returns a proto query or empty proto query if pagination query is empty
func (r *PaginatedQuery) AsV1ScopeQueryOrEmpty() (*v1.Query, error) {
	var q *v1.Query
	if r == nil || r.ScopeQuery == nil {
		q := search.EmptyQuery()
		return q, nil
	}
	q, err := search.ParseQuery(*r.ScopeQuery, search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	return q, nil
}

// String returns a String representation of PaginatedQuery
func (r *PaginatedQuery) String() string {
	if r == nil || r.Query == nil {
		return ""
	}
	return *r.Query
}

// IsEmpty means no query was specified
func (r *PaginatedQuery) IsEmpty() bool {
	return r == nil || r.Query == nil
}

// RawQuery represents a raw query
type RawQuery struct {
	Query      *string
	ScopeQuery *string
}

// AsV1QueryOrEmpty returns a proto query or empty proto query if raw query is empty
func (r RawQuery) AsV1QueryOrEmpty(opts ...search.ParseQueryOption) (*v1.Query, error) {
	if r.Query == nil {
		return search.EmptyQuery(), nil
	}
	opts = append(opts, search.MatchAllIfEmpty())
	return search.ParseQuery(*r.Query, opts...)
}

// AsV1ScopeQueryOrEmpty returns a proto query or empty proto query if pagination query is empty
func (r *RawQuery) AsV1ScopeQueryOrEmpty() (*v1.Query, error) {
	var q *v1.Query
	if r == nil || r.ScopeQuery == nil {
		q := search.EmptyQuery()
		return q, nil
	}
	q, err := search.ParseQuery(*r.ScopeQuery, search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}
	return q, nil
}

// String returns a String representation of RawQuery
func (r RawQuery) String() string {
	if r.Query == nil {
		return ""
	}
	return *r.Query
}

// IsEmpty means no query specified
func (r RawQuery) IsEmpty() bool {
	return r.Query == nil
}

func (resolver *Resolver) getAutoCompleteSearchers() map[v1.SearchCategory]search.Searcher {
	searchers := map[v1.SearchCategory]search.Searcher{
		v1.SearchCategory_ALERTS:                  resolver.ViolationsDataStore,
		v1.SearchCategory_CLUSTERS:                resolver.ClusterDataStore,
		v1.SearchCategory_DEPLOYMENTS:             resolver.DeploymentDataStore,
		v1.SearchCategory_IMAGES:                  resolver.ImageDataStore,
		v1.SearchCategory_POLICIES:                resolver.PolicyDataStore,
		v1.SearchCategory_SECRETS:                 resolver.SecretsDataStore,
		v1.SearchCategory_NAMESPACES:              resolver.NamespaceDataStore,
		v1.SearchCategory_NODES:                   resolver.NodeDataStore,
		v1.SearchCategory_COMPLIANCE:              resolver.ComplianceAggregator,
		v1.SearchCategory_SERVICE_ACCOUNTS:        resolver.ServiceAccountsDataStore,
		v1.SearchCategory_ROLES:                   resolver.K8sRoleStore,
		v1.SearchCategory_ROLEBINDINGS:            resolver.K8sRoleBindingStore,
		v1.SearchCategory_IMAGE_COMPONENTS:        resolver.ImageComponentDataStore,
		v1.SearchCategory_SUBJECTS:                service.NewSubjectSearcher(resolver.K8sRoleBindingStore),
		v1.SearchCategory_IMAGE_VULNERABILITIES:   resolver.ImageCVEDataStore,
		v1.SearchCategory_NODE_VULNERABILITIES:    resolver.NodeCVEDataStore,
		v1.SearchCategory_CLUSTER_VULNERABILITIES: resolver.ClusterCVEDataStore,
		v1.SearchCategory_NODE_COMPONENTS:         resolver.NodeComponentDataStore,
		v1.SearchCategory_POLICY_CATEGORIES:       resolver.PolicyCategoryDataStore,
	}

	return searchers
}

func (resolver *Resolver) getSearchFuncs() map[v1.SearchCategory]searchService.SearchFunc {

	searchfuncs := map[v1.SearchCategory]searchService.SearchFunc{
		v1.SearchCategory_ALERTS:                  resolver.ViolationsDataStore.SearchAlerts,
		v1.SearchCategory_CLUSTERS:                resolver.ClusterDataStore.SearchResults,
		v1.SearchCategory_DEPLOYMENTS:             resolver.DeploymentDataStore.SearchDeployments,
		v1.SearchCategory_IMAGES:                  resolver.ImageDataStore.SearchImages,
		v1.SearchCategory_POLICIES:                resolver.PolicyDataStore.SearchPolicies,
		v1.SearchCategory_SECRETS:                 resolver.SecretsDataStore.SearchSecrets,
		v1.SearchCategory_NAMESPACES:              resolver.NamespaceDataStore.SearchResults,
		v1.SearchCategory_NODES:                   resolver.NodeDataStore.SearchNodes,
		v1.SearchCategory_SERVICE_ACCOUNTS:        resolver.ServiceAccountsDataStore.SearchServiceAccounts,
		v1.SearchCategory_ROLES:                   resolver.K8sRoleStore.SearchRoles,
		v1.SearchCategory_ROLEBINDINGS:            resolver.K8sRoleBindingStore.SearchRoleBindings,
		v1.SearchCategory_IMAGE_COMPONENTS:        resolver.ImageComponentDataStore.SearchImageComponents,
		v1.SearchCategory_SUBJECTS:                service.NewSubjectSearcher(resolver.K8sRoleBindingStore).SearchSubjects,
		v1.SearchCategory_IMAGE_VULNERABILITIES:   resolver.ImageCVEDataStore.SearchImageCVEs,
		v1.SearchCategory_NODE_VULNERABILITIES:    resolver.NodeCVEDataStore.SearchNodeCVEs,
		v1.SearchCategory_CLUSTER_VULNERABILITIES: resolver.ClusterCVEDataStore.SearchClusterCVEs,
		v1.SearchCategory_NODE_COMPONENTS:         resolver.NodeComponentDataStore.SearchNodeComponents,
		v1.SearchCategory_POLICY_CATEGORIES:       resolver.PolicyCategoryDataStore.SearchPolicyCategories,
	}

	return searchfuncs
}

type searchRequest struct {
	Query      string
	Categories *[]string
}

// SearchAutocomplete returns autocomplete responses for the given partial query.
func (resolver *Resolver) SearchAutocomplete(ctx context.Context, args searchRequest) ([]string, error) {
	return searchService.RunAutoComplete(ctx, args.Query, toSearchCategories(args.Categories), resolver.getAutoCompleteSearchers())
}

// SearchOptions gets all search options available for the listed categories
func (resolver *Resolver) SearchOptions(_ context.Context, args struct{ Categories *[]string }) ([]string, error) {
	return searchService.Options(toSearchCategories(args.Categories)), nil
}

// GlobalSearch returns search results for the given categories and query.
// Note: there is not currently a way to request the underlying object from SearchResult; it might be nice to have
// this in the future.
func (resolver *Resolver) GlobalSearch(ctx context.Context, args searchRequest) ([]*searchResultResolver, error) {
	results, _, err := searchService.GlobalSearch(ctx, args.Query, toSearchCategories(args.Categories), resolver.getSearchFuncs())
	return resolver.wrapSearchResults(results, err)
}
