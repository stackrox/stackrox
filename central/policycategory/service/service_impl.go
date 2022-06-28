package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/policycategory/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Policy)): {
			"/v1.PolicyCategoryService/GetPolicyCategory",
			"/v1.PolicyCategoryService/GetPolicyCategories",
		},
		user.With(permissions.Modify(resources.Policy)): {
			"/v1.PolicyCategoryService/PostPolicyCategory",
			"/v1.PolicyCategoryService/RenamePolicyCategory",
			"/v1.PolicyCategoryService/DeletePolicyCategory",
		},
	})
)

var (
	policySyncReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	policyCategoriesDatastore datastore.DataStore
}

func (s *serviceImpl) GetPolicyCategory(ctx context.Context, id *v1.ResourceByID) (*v1.PolicyCategory, error) {
	return s.getPolicyCategory(ctx, id.GetId())
}

func (s *serviceImpl) GetPolicyCategories(ctx context.Context, query *v1.RawQuery) (*v1.GetPolicyCategoriesResponse, error) {
	resp := new(v1.GetPolicyCategoriesResponse)

	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	categories, err := s.policyCategoriesDatastore.SearchRawPolicyCategories(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	resp.Categories = s.convertCategoriesToV1Categories(categories)
	
	return resp, nil
}

func (s *serviceImpl) PostPolicyCategory(ctx context.Context, request *v1.PostPolicyCategoryRequest) (*v1.PolicyCategory, error) {
	category, err := s.policyCategoriesDatastore.AddPolicyCategory(ctx, ToStorageProto(request.GetPolicyCategory()))
	if err != nil {
		return nil, err
	}
	return ToV1Proto(category), nil
}

func (s *serviceImpl) RenamePolicyCategory(ctx context.Context, request *v1.NewRenamePolicyCategoryRequest) (*v1.Empty, error) {
	return &v1.Empty{}, s.policyCategoriesDatastore.RenamePolicyCategory(ctx, request.GetId(), request.GetNewCategoryName())
}

func (s *serviceImpl) DeletePolicyCategory(ctx context.Context, request *v1.NewDeletePolicyCategoryRequest) (*v1.Empty, error) {
	return &v1.Empty{}, s.policyCategoriesDatastore.DeletePolicyCategory(ctx, request.GetId())

}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPolicyCategoryServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPolicyServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) getPolicyCategory(ctx context.Context, id string) (*v1.PolicyCategory, error) {
	if id == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Policy category ID must be provided")
	}
	category, exists, err := s.policyCategoriesDatastore.GetPolicyCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "policy with ID '%s' does not exist", id)
	}
	return ToV1Proto(category), nil

}

func (s *serviceImpl) convertCategoriesToV1Categories(categories []*storage.PolicyCategory) []*v1.PolicyCategory {
	v1Categories := make([]*v1.PolicyCategory, len(categories))
	for _, category := range categories {
		v1Categories = append(v1Categories, ToV1Proto(category))
	}
	return v1Categories
}
