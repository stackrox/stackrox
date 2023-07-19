package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/alert/mappings"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	complianceSearch "github.com/stackrox/rox/central/compliance/search"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex/mapping"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	categoriesDataStore "github.com/stackrox/rox/central/policycategory/datastore"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/rbac/service"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	centralsearch "github.com/stackrox/rox/central/search"
	"github.com/stackrox/rox/central/search/options"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
)

const maxAutocompleteResults = 10

var (
	categoryToOptionsMultimap = func() map[v1.SearchCategory]search.OptionsMultiMap {
		result := make(map[v1.SearchCategory]search.OptionsMultiMap)
		for cat, optMap := range mapping.GetEntityOptionsMap() {
			result[cat] = search.MultiMapFromMaps(optMap)
		}
		result[v1.SearchCategory_COMPLIANCE] = complianceSearch.SearchOptionsMultiMap
		return result
	}()
)

type autocompleteResult struct {
	value string
	score float64
}

// SearchFunc represents a function that goes from a query to a proto search result.
type SearchFunc func(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

func (s *serviceImpl) getSearchFuncs() map[v1.SearchCategory]SearchFunc {
	searchfuncs := map[v1.SearchCategory]SearchFunc{
		v1.SearchCategory_ALERTS:             s.alerts.SearchAlerts,
		v1.SearchCategory_DEPLOYMENTS:        s.deployments.SearchDeployments,
		v1.SearchCategory_IMAGES:             s.images.SearchImages,
		v1.SearchCategory_POLICIES:           s.policies.SearchPolicies,
		v1.SearchCategory_SECRETS:            s.secrets.SearchSecrets,
		v1.SearchCategory_NAMESPACES:         s.namespaces.SearchResults,
		v1.SearchCategory_NODES:              s.nodes.SearchNodes,
		v1.SearchCategory_CLUSTERS:           s.clusters.SearchResults,
		v1.SearchCategory_SERVICE_ACCOUNTS:   s.serviceaccounts.SearchServiceAccounts,
		v1.SearchCategory_ROLES:              s.roles.SearchRoles,
		v1.SearchCategory_ROLEBINDINGS:       s.bindings.SearchRoleBindings,
		v1.SearchCategory_SUBJECTS:           service.NewSubjectSearcher(s.bindings).SearchSubjects,
		v1.SearchCategory_IMAGE_INTEGRATIONS: s.imageIntegrations.SearchImageIntegrations,
		v1.SearchCategory_POLICY_CATEGORIES:  s.categories.SearchPolicyCategories,
	}

	return searchfuncs
}

func (s *serviceImpl) getAutocompleteSearchers() map[v1.SearchCategory]search.Searcher {
	searchers := map[v1.SearchCategory]search.Searcher{
		v1.SearchCategory_ALERTS:             s.alerts,
		v1.SearchCategory_DEPLOYMENTS:        s.deployments,
		v1.SearchCategory_IMAGES:             s.images,
		v1.SearchCategory_POLICIES:           s.policies,
		v1.SearchCategory_SECRETS:            s.secrets,
		v1.SearchCategory_NAMESPACES:         s.namespaces,
		v1.SearchCategory_NODES:              s.nodes,
		v1.SearchCategory_COMPLIANCE:         s.aggregator,
		v1.SearchCategory_RISKS:              s.risks,
		v1.SearchCategory_CLUSTERS:           s.clusters,
		v1.SearchCategory_SERVICE_ACCOUNTS:   s.serviceaccounts,
		v1.SearchCategory_ROLES:              s.roles,
		v1.SearchCategory_ROLEBINDINGS:       s.bindings,
		v1.SearchCategory_SUBJECTS:           service.NewSubjectSearcher(s.bindings),
		v1.SearchCategory_IMAGE_INTEGRATIONS: s.imageIntegrations,
		v1.SearchCategory_POLICY_CATEGORIES:  s.categories,
	}

	return searchers
}

var (
	autocompleteCategories = func() set.Set[v1.SearchCategory] {
		s := centralsearch.GetGlobalSearchCategories().Clone()
		s.Add(v1.SearchCategory_COMPLIANCE)
		return s
	}()
)

// SearchService provides APIs for search.
type serviceImpl struct {
	v1.UnimplementedSearchServiceServer

	alerts            alertDataStore.DataStore
	deployments       deploymentDataStore.DataStore
	images            imageDataStore.DataStore
	policies          policyDataStore.DataStore
	secrets           secretDataStore.DataStore
	serviceaccounts   serviceAccountDataStore.DataStore
	nodes             nodeDataStore.DataStore
	namespaces        namespaceDataStore.DataStore
	risks             riskDataStore.DataStore
	roles             roleDataStore.DataStore
	bindings          roleBindingDataStore.DataStore
	clusters          clusterDataStore.DataStore
	categories        categoriesDataStore.DataStore
	aggregator        aggregation.Aggregator
	authorizer        authz.Authorizer
	imageIntegrations imageIntegrationDataStore.DataStore
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

func trimMatches(matches map[string][]string, fieldPaths []string) map[string][]string {
	result := make(map[string][]string, len(fieldPaths))
	for _, fp := range fieldPaths {
		vals, ok := matches[fp]
		if ok {
			result[fp] = vals
		}
	}
	return result
}

// RunAutoComplete runs an autocomplete request. It's a free function used by both regular search and by GraphQL.
func RunAutoComplete(ctx context.Context, queryString string, categories []v1.SearchCategory, searchers map[v1.SearchCategory]search.Searcher) ([]string, error) {
	query, autocompleteKey, err := search.ParseQueryForAutocomplete(queryString)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "unable to parse query %q: %v", queryString, err)
	}
	// Set the max return size for the query
	query.Pagination = &v1.QueryPagination{
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
		if searcher == nil {
			if ok {
				utils.Should(errors.Errorf("searchers map has an entry for category %v, but the returned searcher was nil", category))
			}
			return nil, errors.Wrapf(errox.InvalidArgs, "Search category %q is not implemented", category.String())
		}

		optMultiMap := categoryToOptionsMultimap[category]
		if optMultiMap == nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "Search category %q is not implemented", category.String())
		}

		autocompleteFields := optMultiMap.GetAll(autocompleteKey)
		if len(autocompleteFields) == 0 {
			// Category for field to be autocompleted not applicable.
			continue
		}

		// All the field paths to consider for the autocomplete field.
		fieldPaths := make([]string, 0, 3*len(autocompleteFields))
		for _, field := range autocompleteFields {
			fieldPaths = append(fieldPaths,
				field.GetFieldPath(),
				search.ToMapKeyPath(field.GetFieldPath()),
				search.ToMapValuePath(field.GetFieldPath()),
			)
		}

		results, err := searcher.Search(ctx, query)
		if err != nil {
			log.Errorf("failed to search category %s: %s", category.String(), err)
			return nil, err
		}
		for _, r := range results {
			matches := trimMatches(r.Matches, fieldPaths)
			// In postgres, we do not need to combine map key and values matches as `k=v` because it is already done by postgres searcher.
			// With postgres, the following condition will not pass anyway.
			//
			// This implies that the object is a map because it has multiple values
			if isMapMatch(matches) {
				autocompleteResults = append(autocompleteResults, handleMapResults(matches, r.Score)...)
				continue
			}

			for fieldPath, match := range matches {
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

func (s *serviceImpl) autocomplete(ctx context.Context, queryString string, categories []v1.SearchCategory) ([]string, error) {
	return RunAutoComplete(ctx, queryString, categories, s.getAutocompleteSearchers())
}

func (s *serviceImpl) Autocomplete(ctx context.Context, req *v1.RawSearchRequest) (*v1.AutocompleteResponse, error) {
	if req.GetQuery() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "query cannot be empty")
	}
	results, err := s.autocomplete(ctx, req.GetQuery(), req.GetCategories())
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
	s.authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.Authenticated(): {
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
func GlobalSearch(ctx context.Context, query string, categories []v1.SearchCategory, searchFuncMap map[v1.SearchCategory]SearchFunc) (results []*v1.SearchResult,
	counts []*v1.SearchResponse_Count, err error) {

	parsedRequest, err := search.ParseQuery(query)
	if err != nil {
		err = errors.Wrap(errox.InvalidArgs, err.Error())
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
			err = errors.Wrapf(errox.InvalidArgs, "Search category '%s' is not implemented", category.String())
			return
		}
		var resultsFromCategory []*v1.SearchResult
		resultsFromCategory, err = searchFunc(ctx, parsedRequest)
		if err != nil {
			log.Errorf("error searching for %s: %v", category, err)
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
	results, counts, err := GlobalSearch(ctx, request.GetQuery(), request.GetCategories(), s.getSearchFuncs())
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
func (s *serviceImpl) Options(_ context.Context, request *v1.SearchOptionsRequest) (*v1.SearchOptionsResponse, error) {
	return &v1.SearchOptionsResponse{Options: Options(request.GetCategories())}, nil
}

// GetAllSearchableCategories returns a list of categories that are currently valid for global search
func GetAllSearchableCategories() (categories []v1.SearchCategory) {
	return centralsearch.GetGlobalSearchCategories().AsSortedSlice(func(catI, catJ v1.SearchCategory) bool {
		return catI < catJ
	})
}
