package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/pod/datastore"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPodsReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			"/v1.PodService/GetPods",
		},
	})
)

// serviceImpl provides APIs for deployments.
type serviceImpl struct {
	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPodServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPodServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetPods returns Pods according to the request.
func (s *serviceImpl) GetPods(ctx context.Context, request *v1.RawQuery) (*v1.PodsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.GetPagination(), maxPodsReturned)

	pods, err := s.datastore.SearchRawPods(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}

	return &v1.PodsResponse{
		Pods: pods,
	}, nil
}
