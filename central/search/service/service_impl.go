package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/deckarep/golang-set"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/alert/index/mappings"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/search/options"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
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

type searchFunc func(q *v1.Query) ([]*v1.SearchResult, error)

func (s *serviceImpl) getSearchFuncs() map[v1.SearchCategory]searchFunc {
	return map[v1.SearchCategory]searchFunc{
		v1.SearchCategory_ALERTS:      s.alerts.SearchAlerts,
		v1.SearchCategory_DEPLOYMENTS: s.deployments.SearchDeployments,
		v1.SearchCategory_IMAGES:      s.images.SearchImages,
		v1.SearchCategory_POLICIES:    s.policies.SearchPolicies,
		v1.SearchCategory_SECRETS:     s.secrets.SearchSecrets,
	}
}

type autocompleteFunc func(q *v1.Query) ([]search.Result, error)

func (s *serviceImpl) getAutocompleteFuncs() map[v1.SearchCategory]autocompleteFunc {
	return map[v1.SearchCategory]autocompleteFunc{
		v1.SearchCategory_ALERTS:      s.alerts.Search,
		v1.SearchCategory_DEPLOYMENTS: s.deployments.Search,
		v1.SearchCategory_IMAGES:      s.images.Search,
		v1.SearchCategory_POLICIES:    s.policies.Search,
		v1.SearchCategory_SECRETS:     s.secrets.Search,
	}
}

var (
	// GlobalSearchCategories is exposed for e2e options test
	GlobalSearchCategories = mapset.NewSet(
		v1.SearchCategory_ALERTS,
		v1.SearchCategory_DEPLOYMENTS,
		v1.SearchCategory_IMAGES,
		v1.SearchCategory_POLICIES,
		v1.SearchCategory_SECRETS,
	)

	// To access search, we require users to have view access to every searchable resource.
	// We could consider allowing people to search across just the things they have access to,
	// but that requires non-trivial refactoring, so we'll do it if we feel the need later.
	// This variable is package-level to facilitate the unit test that asserts
	// that it covers all the searchable categories.
	searchCategoryToResource = map[v1.SearchCategory]permissions.Resource{
		v1.SearchCategory_ALERTS:      resources.Alert,
		v1.SearchCategory_DEPLOYMENTS: resources.Deployment,
		v1.SearchCategory_IMAGES:      resources.Image,
		v1.SearchCategory_POLICIES:    resources.Policy,
		v1.SearchCategory_SECRETS:     resources.Secret,
	}
)

// SearchService provides APIs for search.
type serviceImpl struct {
	alerts       alertDataStore.DataStore
	deployments  deploymentDataStore.DataStore
	images       imageDataStore.DataStore
	policies     policyDataStore.DataStore
	secrets      secretDataStore.DataStore
	enumRegistry enumregistry.Registry

	authorizer authz.Authorizer
}

func (s *serviceImpl) handleMatch(fieldPath, value string, isEnum bool) string {
	if !isEnum {
		return value
	}
	if val, err := strconv.ParseInt(value, 10, 32); err == nil {
		// Lookup if the field path is an enum and if so, take the string representation
		if enumString := s.enumRegistry.Lookup(fieldPath, int32(val)); enumString != "" {
			return enumString
		}
	}
	return value
}

func (s *serviceImpl) autocomplete(queryString string, categories []v1.SearchCategory) ([]string, error) {
	query, err := search.ParseAutocompleteRawQuery(queryString)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to parse query %q: %v", queryString, err)
	}
	// Set the max return size for the query
	query.MaxResultSize = maxAutocompleteResults

	searchFuncMap := s.getAutocompleteFuncs()
	if len(categories) == 0 {
		categories = GetAllSearchableCategories()
	}
	var autocompleteResults []autocompleteResult
	for _, category := range categories {
		if category == v1.SearchCategory_ALERTS && !shouldProcessAlerts(query) {
			continue
		}
		searchFunc, ok := searchFuncMap[category]
		if !ok {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Search category '%s' is not implemented", category.String()))
		}
		results, err := searchFunc(query)
		if err != nil {
			log.Error(err)
			return nil, status.Error(codes.Internal, err.Error())
		}
		for _, r := range results {
			for fieldPath, match := range r.Matches {
				isEnum := s.enumRegistry.IsEnum(fieldPath)
				for _, v := range match {
					value := s.handleMatch(fieldPath, v, isEnum)
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

// Search implements the ability to search through indexes for data
func (s *serviceImpl) Search(ctx context.Context, request *v1.RawSearchRequest) (*v1.SearchResponse, error) {
	parsedRequest, err := search.ParseRawQuery(request.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	response := new(v1.SearchResponse)
	searchFuncMap := s.getSearchFuncs()
	categories := request.GetCategories()
	if len(categories) == 0 {
		categories = GetAllSearchableCategories()
	}
	for _, category := range categories {
		if category == v1.SearchCategory_ALERTS && !shouldProcessAlerts(parsedRequest) {
			response.Counts = append(response.Counts, &v1.SearchResponse_Count{Category: category, Count: 0})
			continue
		}
		searchFunc, ok := searchFuncMap[category]
		if !ok {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Search category '%s' is not implemented", category.String()))
		}
		results, err := searchFunc(parsedRequest)
		if err != nil {
			log.Error(err)
			return nil, status.Error(codes.Internal, err.Error())
		}
		response.Counts = append(response.Counts, &v1.SearchResponse_Count{Category: category, Count: int64(len(results))})
		response.Results = append(response.Results, results...)
	}
	// Sort from highest score to lowest
	sort.SliceStable(response.Results, func(i, j int) bool { return response.Results[i].Score > response.Results[j].Score })
	return response, nil
}

// Options returns the options available for the categories specified in the request
func (s *serviceImpl) Options(ctx context.Context, request *v1.SearchOptionsRequest) (*v1.SearchOptionsResponse, error) {
	categories := request.GetCategories()
	if len(categories) == 0 {
		categories = GetAllSearchableCategories()
	}
	return &v1.SearchOptionsResponse{
		Options: options.GetOptions(categories),
	}, nil
}

// GetAllSearchableCategories returns a list of categories that are currently valid for global search
func GetAllSearchableCategories() (categories []v1.SearchCategory) {
	categories = make([]v1.SearchCategory, 0, len(v1.SearchCategory_name)-1)
	for i := 1; i < len(v1.SearchCategory_name); i++ {
		category := v1.SearchCategory(i)
		// For now, process indicators are not covered in global search.
		if !GlobalSearchCategories.Contains(category) {
			continue
		}
		categories = append(categories, category)
	}
	return
}
