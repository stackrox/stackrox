package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/pod/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPodsReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			v1.PodService_GetPods_FullMethodName,
			v1.PodService_ExportPods_FullMethodName,
		},
	})
)

// serviceImpl provides APIs for deployments.
type serviceImpl struct {
	v1.UnimplementedPodServiceServer

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

func (s *serviceImpl) ExportPods(req *v1.ExportPodRequest, srv v1.PodService_ExportPodsServer) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	ctx := srv.Context()
	if timeout := req.GetTimeout(); timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(srv.Context(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	return s.datastore.WalkByQuery(ctx, parsedQuery, func(p *storage.Pod) error {
		if err := srv.Send(&v1.ExportPodResponse{Pod: p}); err != nil {
			return err
		}
		return nil
	})
}
