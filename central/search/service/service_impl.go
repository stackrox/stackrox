package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/search/options"
	secretService "github.com/stackrox/rox/central/secret/service"
	"github.com/stackrox/rox/central/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	searchCommon "github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type searchFunc func(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)

func (s *serviceImpl) getSearchFuncs() map[v1.SearchCategory]searchFunc {
	return map[v1.SearchCategory]searchFunc{
		v1.SearchCategory_ALERTS:      s.alerts.SearchAlerts,
		v1.SearchCategory_DEPLOYMENTS: s.deployments.SearchDeployments,
		v1.SearchCategory_IMAGES:      s.images.SearchImages,
		v1.SearchCategory_POLICIES:    s.policies.SearchPolicies,
		v1.SearchCategory_SECRETS:     secretService.ParsedSearchRequestHandler(),
	}
}

var (
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
	alerts      alertDataStore.DataStore
	deployments deploymentDataStore.DataStore
	images      imageDataStore.DataStore
	policies    policyDataStore.DataStore

	parser *searchCommon.QueryParser

	authorizer authz.Authorizer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSearchServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterSearchServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (s *serviceImpl) initializeAuthorizer() {
	requiredPermissions := make([]permissions.Permission, 0, len(searchCategoryToResource))
	for _, resource := range searchCategoryToResource {
		requiredPermissions = append(requiredPermissions, permissions.View(resource))
	}

	s.authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(requiredPermissions...): {
			"/v1.SearchService/Search",
			"/v1.SearchService/Options",
		},
	})
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(s.authorizer.Authorized(ctx, fullMethodName))
}

// Search implements the ability to search through indexes for data
func (s *serviceImpl) Search(ctx context.Context, request *v1.RawSearchRequest) (*v1.SearchResponse, error) {
	parsedRequest, err := s.parser.ParseRawQuery(request.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	response := new(v1.SearchResponse)
	searchFuncMap := s.getSearchFuncs()
	categories := request.GetCategories()
	if len(categories) == 0 {
		categories = getAllCategories()
	}
	for _, category := range categories {
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
		categories = getAllCategories()
	}
	return &v1.SearchOptionsResponse{
		Options: options.GetOptions(categories),
	}, nil
}

func getAllCategories() (categories []v1.SearchCategory) {
	categories = make([]v1.SearchCategory, 0, len(v1.SearchCategory_name)-1)
	for i := 1; i < len(v1.SearchCategory_name); i++ {
		categories = append(categories, v1.SearchCategory(i))
	}
	return
}
