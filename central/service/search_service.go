package service

import (
	"context"
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewSearchService returns the SearchService object.
func NewSearchService(indexer search.Indexer) *SearchService {
	return &SearchService{
		indexer: indexer,
	}
}

// SearchService provides APIs for search.
type SearchService struct {
	indexer search.Indexer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *SearchService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSearchServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *SearchService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterSearchServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *SearchService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(user.Any().Authorized(ctx))
}

type searchFunc func(request *v1.SearchRequest) ([]string, error)

func (s *SearchService) getSearchFuncs() map[v1.SearchCategory]searchFunc {
	return map[v1.SearchCategory]searchFunc{
		v1.SearchCategory_ALERTS:      s.indexer.SearchAlerts,
		v1.SearchCategory_DEPLOYMENTS: s.indexer.SearchDeployments,
		v1.SearchCategory_IMAGES:      s.indexer.SearchImages,
		v1.SearchCategory_POLICIES:    s.indexer.SearchPolicies,
	}
}

func getAllCategories() (categories []v1.SearchCategory) {
	categories = make([]v1.SearchCategory, 0, len(v1.SearchCategory_name)-1)
	for i := 1; i < len(v1.SearchCategory_name); i++ {
		categories = append(categories, v1.SearchCategory(i))
	}
	return
}

func validateRequest(request *v1.SearchRequest) error {
	for field, values := range request.GetFields() {
		if len(values.GetValues()) == 0 {
			return fmt.Errorf("Field '%s' must have at least 1 value", field)
		}
	}
	return nil
}

// Search implements the ability to search through indexes for data
func (s *SearchService) Search(ctx context.Context, request *v1.SearchRequest) (*v1.SearchResponse, error) {
	if err := validateRequest(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	response := new(v1.SearchResponse)
	searchFuncMap := s.getSearchFuncs()
	for _, category := range request.Categories {
		f, ok := searchFuncMap[category]
		if !ok {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Search category '%s' is not implemented", category.String()))
		}
		ids, err := f(request)
		if err != nil {
			log.Error(err)
			return nil, status.Error(codes.Internal, err.Error())
		}
		response.Results = append(response.Results, &v1.SearchResult{Category: category, Ids: ids})
	}
	return response, nil
}

// Options returns the options available for the categories specified in the request
func (s *SearchService) Options(ctx context.Context, request *v1.SearchOptionsRequest) (*v1.SearchOptionsResponse, error) {
	categories := request.GetCategories()
	if len(categories) == 0 {
		categories = getAllCategories()
	}
	return &v1.SearchOptionsResponse{
		Options: search.GetOptions(categories),
	}, nil
}
