package service

import (
	"context"
	"fmt"
	"sort"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	policyDataStore "bitbucket.org/stack-rox/apollo/central/policy/datastore"
	"bitbucket.org/stack-rox/apollo/central/search/options"
	secretService "bitbucket.org/stack-rox/apollo/central/secret/service"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	searchCommon "bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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

// SearchService provides APIs for search.
type serviceImpl struct {
	alerts      alertDataStore.DataStore
	deployments deploymentDataStore.DataStore
	images      imageDataStore.DataStore
	policies    policyDataStore.DataStore

	parser *searchCommon.QueryParser
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSearchServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterSearchServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(user.Any().Authorized(ctx))
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
