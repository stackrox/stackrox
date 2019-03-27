package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/alert/index/mappings"
	"github.com/stackrox/rox/central/compliance/aggregation"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globalstore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/search/options"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxAutocompleteResults = 10

type autocompleteResult struct {
	value string
	score float64
}

// SearchFunc represents a function that goes from a query to a proto search result.
type SearchFunc func(q *v1.Query) ([]*v1.SearchResult, error)

func (s *serviceImpl) getSearchFuncs() map[v1.SearchCategory]SearchFunc {
	searchfuncs := map[v1.SearchCategory]SearchFunc{
		v1.SearchCategory_ALERTS:      s.alerts.SearchAlerts,
		v1.SearchCategory_DEPLOYMENTS: s.deployments.SearchDeployments,
		v1.SearchCategory_IMAGES:      s.images.SearchImages,
		v1.SearchCategory_POLICIES:    s.policies.SearchPolicies,
		v1.SearchCategory_SECRETS:     s.secrets.SearchSecrets,
		v1.SearchCategory_NAMESPACES:  s.namespaces.SearchResults,
		v1.SearchCategory_NODES:       s.nodes.SearchResults,
	}

	if features.K8sRBAC.Enabled() {
		searchfuncs[v1.SearchCategory_SERVICE_ACCOUNTS] = s.serviceAccounts.SearchServiceAccounts
	}

	return searchfuncs
}

func (s *serviceImpl) getAutocompleteSearchers() map[v1.SearchCategory]search.Searcher {
	searchers := map[v1.SearchCategory]search.Searcher{
		v1.SearchCategory_ALERTS:      s.alerts,
		v1.SearchCategory_DEPLOYMENTS: s.deployments,
		v1.SearchCategory_IMAGES:      s.images,
		v1.SearchCategory_POLICIES:    s.policies,
		v1.SearchCategory_SECRETS:     s.secrets,
		v1.SearchCategory_NAMESPACES:  s.namespaces,
		v1.SearchCategory_NODES:       s.nodes,
		v1.SearchCategory_COMPLIANCE:  s.aggregator,
	}

	if features.K8sRBAC.Enabled() {
		searchers[v1.SearchCategory_SERVICE_ACCOUNTS] = s.serviceAccounts
	}

	return searchers
}

var (
	autocompleteCategories = func() set.V1SearchCategorySet {
		s := set.NewV1SearchCategorySet(GetGlobalSearchCategories().AsSlice()...)
		s.Add(v1.SearchCategory_COMPLIANCE)
		return s
	}()
)

// GetSearchCategoryToResource gets a map of search category to corresponding resource
func GetSearchCategoryToResource() map[v1.SearchCategory]permissions.Resource {

	// SearchCategoryToResource maps search categories to resources.
	// To access search, we require users to have view access to every searchable resource.
	// We could consider allowing people to search across just the things they have access to,
	// but that requires non-trivial refactoring, so we'll do it if we feel the need later.
	// This variable is package-level to facilitate the unit test that asserts
	// that it covers all the searchable categories.
	searchCategoryToResource := map[v1.SearchCategory]permissions.Resource{
		v1.SearchCategory_ALERTS:      resources.Alert,
		v1.SearchCategory_DEPLOYMENTS: resources.Deployment,
		v1.SearchCategory_IMAGES:      resources.Image,
		v1.SearchCategory_POLICIES:    resources.Policy,
		v1.SearchCategory_SECRETS:     resources.Secret,
		v1.SearchCategory_COMPLIANCE:  resources.Compliance,
		v1.SearchCategory_NODES:       resources.Node,
		v1.SearchCategory_NAMESPACES:  resources.Namespace,
	}

	if features.K8sRBAC.Enabled() {
		searchCategoryToResource[v1.SearchCategory_SERVICE_ACCOUNTS] = resources.ServiceAccount
	}

	return searchCategoryToResource
}

// GetGlobalSearchCategories returns a set of search categories
func GetGlobalSearchCategories() set.V1SearchCategorySet {
	// globalSearchCategories is exposed for e2e options test
	globalSearchCategories := set.NewV1SearchCategorySet(
		v1.SearchCategory_ALERTS,
		v1.SearchCategory_DEPLOYMENTS,
		v1.SearchCategory_IMAGES,
		v1.SearchCategory_POLICIES,
		v1.SearchCategory_SECRETS,
		v1.SearchCategory_NODES,
		v1.SearchCategory_NAMESPACES,
	)

	if features.K8sRBAC.Enabled() {
		globalSearchCategories.Add(v1.SearchCategory_SERVICE_ACCOUNTS)
	}

	return globalSearchCategories

}

// SearchService provides APIs for search.
type serviceImpl struct {
	alerts          alertDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	images          imageDataStore.DataStore
	policies        policyDataStore.DataStore
	secrets         secretDataStore.DataStore
	serviceAccounts serviceAccountDataStore.DataStore
	nodes           nodeDataStore.GlobalStore
	namespaces      namespaceDataStore.DataStore

	aggregator aggregation.Aggregator
	authorizer authz.Authorizer
}

func handleMatch(fieldPath, value string) string {
	if !enumregistry.IsEnum(fieldPath) {
		return value
	}
	if val, err := strconv.ParseInt(value, 10, 32); err == nil {
		// Lookup if the field path is an enum and if so, take the string representation
		if enumString := enumregistry.Lookup(fieldPath, int32(val)); enumString != "" {
			return enumString
		}
	}
	return value
}

func handleMapResults(matches map[string][]string, score float64) []autocompleteResult {
	var keys []string
	var values []string
	for k, match := range matches {
		if strings.HasSuffix(k, "key") {
			keys = match
		} else {
			values = match
		}
	}
	results := make([]autocompleteResult, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		results = append(results, autocompleteResult{value: fmt.Sprintf("%s=%s", keys[i], values[i]), score: score})
	}
	return results
}

func isMapMatch(matches map[string][]string) bool {
	for k := range matches {
		if !strings.HasSuffix(k, ".keypair.key") && !strings.HasSuffix(k, ".keypair.value") {
			return false
		}
	}
	return true
}

// RunAutoComplete runs an autocomplete request. It's a free function used by both regular search and by GraphQL.
func RunAutoComplete(queryString string, categories []v1.SearchCategory, searchers map[v1.SearchCategory]search.Searcher) ([]string, error) {
	query, err := search.ParseAutocompleteRawQuery(queryString)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to parse query %q: %v", queryString, err)
	}
	// Set the max return size for the query
	query.Pagination = &v1.Pagination{
		Limit: maxAutocompleteResults,
	}

	if len(categories) == 0 {
		categories = autocompleteCategories.AsSlice()
	}
	var autocompleteResults []autocompleteResult
	for _, category := range categories {
		if category == v1.SearchCategory_ALERTS && !shouldProcessAlerts(query) {
			continue
		}
		searcher, ok := searchers[category]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "Search category '%s' is not implemented", category.String())
		}
		results, err := searcher.Search(query)
		if err != nil {
			log.Error(err)
			return nil, status.Error(codes.Internal, err.Error())
		}
		for _, r := range results {
			// This implies that the object is a map because it has multiple values
			if isMapMatch(r.Matches) {
				autocompleteResults = append(autocompleteResults, handleMapResults(r.Matches, r.Score)...)
				continue
			}
			for fieldPath, match := range r.Matches {
				for _, v := range match {
					value := handleMatch(fieldPath, v)
					autocompleteResults = append(autocompleteResults, autocompleteResult{value: value, score: r.Score})
				}
			}
		}
	}

	sort.Slice(autocompleteResults, func(i, j int) bool { return autocompleteResults[i].score > autocompleteResults[j].score })
	resultSet := set.NewStringSet()

	var stringResults []string
	for _, a := range autocompleteResults {
		if added := resultSet.Add(a.value); added {
			stringResults = append(stringResults, a.value)
		}
		if resultSet.Cardinality() == maxAutocompleteResults {
			break
		}
	}
	return stringResults, nil
}

func (s *serviceImpl) autocomplete(queryString string, categories []v1.SearchCategory) ([]string, error) {
	return RunAutoComplete(queryString, categories, s.getAutocompleteSearchers())
}

func (s *serviceImpl) Autocomplete(ctx context.Context, req *v1.RawSearchRequest) (*v1.AutocompleteResponse, error) {
	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "query cannot be empty")
	}
	results, err := s.autocomplete(req.GetQuery(), req.GetCategories())
	if err != nil {
		return nil, err
	}
	return &v1.AutocompleteResponse{Values: results}, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSearchServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSearchServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) initializeAuthorizer() {
	searchCategoryToResource := GetSearchCategoryToResource()
	requiredPermissions := make([]*v1.Permission, 0, len(searchCategoryToResource))
	for _, resource := range searchCategoryToResource {
		requiredPermissions = append(requiredPermissions, permissions.View(resource))
	}

	s.authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(requiredPermissions...): {
			"/v1.SearchService/Search",
			"/v1.SearchService/Options",
			"/v1.SearchService/Autocomplete",
		},
	})
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, s.authorizer.Authorized(ctx, fullMethodName)
}

// Special case alerts because they have a default search param of state:unresolved
// TODO(cgorman) rework the options for global search to allow for transitive connections (policy <-> deployment, etc)
func shouldProcessAlerts(q *v1.Query) (shouldProcess bool) {
	fn := func(bq *v1.BaseQuery) {
		mfq, ok := bq.Query.(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if _, ok := mappings.OptionsMap.Get(mfq.MatchFieldQuery.Field); ok {
			shouldProcess = true
		}
	}
	search.ApplyFnToAllBaseQueries(q, fn)
	return
}

// GlobalSearch runs a global search request with the given arguments. It's a shared function between gRPC and GraphQL.
func GlobalSearch(query string, categories []v1.SearchCategory, searchFuncMap map[v1.SearchCategory]SearchFunc) (results []*v1.SearchResult,
	counts []*v1.SearchResponse_Count, err error) {

	parsedRequest, err := search.ParseRawQuery(query)
	if err != nil {
		err = status.Error(codes.InvalidArgument, err.Error())
		return
	}
	if len(categories) == 0 {
		categories = GetAllSearchableCategories()
	}
	for _, category := range categories {
		if category == v1.SearchCategory_ALERTS && !shouldProcessAlerts(parsedRequest) {
			counts = append(counts, &v1.SearchResponse_Count{Category: category, Count: 0})
			continue
		}
		searchFunc, ok := searchFuncMap[category]
		if !ok {
			err = status.Error(codes.InvalidArgument, fmt.Sprintf("Search category '%s' is not implemented", category.String()))
			return
		}
		var resultsFromCategory []*v1.SearchResult
		resultsFromCategory, err = searchFunc(parsedRequest)
		if err != nil {
			log.Error(err)
			err = status.Error(codes.Internal, err.Error())
			return
		}
		counts = append(counts, &v1.SearchResponse_Count{Category: category, Count: int64(len(resultsFromCategory))})
		results = append(results, resultsFromCategory...)
	}
	// Sort from highest score to lowest
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return
}

// Search implements the ability to search through indexes for data
func (s *serviceImpl) Search(ctx context.Context, request *v1.RawSearchRequest) (*v1.SearchResponse, error) {
	results, counts, err := GlobalSearch(request.GetQuery(), request.GetCategories(), s.getSearchFuncs())
	if err != nil {
		return nil, err
	}
	return &v1.SearchResponse{
		Results: results,
		Counts:  counts,
	}, nil
}

// Options returns the list of options for the given categories, defaulting to all searchable categories
// if not specified. It is shared between gRPC and GraphQL.
func Options(categories []v1.SearchCategory) []string {
	if len(categories) == 0 {
		categories = GetAllSearchableCategories()
	}
	return options.GetOptions(categories)
}

// Options returns the options available for the categories specified in the request
func (s *serviceImpl) Options(ctx context.Context, request *v1.SearchOptionsRequest) (*v1.SearchOptionsResponse, error) {
	return &v1.SearchOptionsResponse{Options: Options(request.GetCategories())}, nil
}

// GetAllSearchableCategories returns a list of categories that are currently valid for global search
func GetAllSearchableCategories() (categories []v1.SearchCategory) {
	return GetGlobalSearchCategories().AsSortedSlice(func(catI, catJ v1.SearchCategory) bool {
		return catI < catJ
	})
}
