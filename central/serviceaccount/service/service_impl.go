package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ServiceAccount)): {
			"/v1.ServiceAccountService/GetServiceAccount",
			"/v1.ServiceAccountService/ListServiceAccounts",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	storage datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterServiceAccountServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterServiceAccountServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetServiceAccount returns the service account for the id.
func (s *serviceImpl) GetServiceAccount(ctx context.Context, request *v1.ResourceByID) (*v1.GetServiceAccountResponse, error) {
	sa, exists, err := s.storage.GetServiceAccount(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "service account with id '%s' does not exist", request.GetId())
	}
	return &v1.GetServiceAccountResponse{
		ServiceAccount: sa,
	}, nil
}

// ListServiceAccounts returns all service accounts that match the query.
func (s *serviceImpl) ListServiceAccounts(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListServiceAccountResponse, error) {
	var serviceAccounts []*storage.ServiceAccount
	var err error
	if rawQuery.GetQuery() == "" {
		serviceAccounts, err = s.storage.ListServiceAccounts()
	} else {
		var q *v1.Query
		q, err = search.ParseRawQueryOrEmpty(rawQuery.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		serviceAccounts, err = s.storage.SearchRawServiceAccounts(q)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve service accounts: %s", err)
	}

	return &v1.ListServiceAccountResponse{ServiceAccounts: serviceAccounts}, nil
}
