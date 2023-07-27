package usage

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/usage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.UsageService/GetCurrentUsage",
			"/v1.UsageService/GetMaxUsage",
		}})
)

type serviceImpl struct {
	v1.UnimplementedUsageServiceServer

	datastore datastore.DataStore
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterUsageServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return errors.Wrap(v1.RegisterUsageServiceHandler(ctx, mux, conn), "failed to register the usage service handler")
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, errors.Wrapf(authorizer.Authorized(ctx, fullMethodName), "failed to authorize a call to %s", fullMethodName)
}

func (s *serviceImpl) GetCurrentUsage(ctx context.Context, _ *v1.Empty) (*v1.CurrentUsageResponse, error) {
	current := &v1.CurrentUsageResponse{
		Timestamp: protoconv.ConvertTimeToTimestamp(time.Now().UTC())}
	if m, err := s.datastore.GetCurrent(ctx); err != nil {
		return nil, errors.Wrap(err, "datastore failed to get current usage metrics")
	} else if m != nil && m.Sr != nil {
		current.NumNodes = m.Sr.Nodes
		current.NumCores = m.Sr.Cores
	}
	return current, nil
}

func (s *serviceImpl) GetMaxUsage(ctx context.Context, req *v1.UsageRequest) (*v1.MaxUsageResponse, error) {
	metrics, err := s.datastore.Get(ctx, req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, errors.Wrap(err, "cannot get usage")
	}
	max := &v1.MaxUsageResponse{}
	for _, m := range metrics {
		if n := m.GetSr().GetNodes(); n >= max.MaxNodes {
			max.MaxNodes = n
			max.MaxNodesAt = m.GetTs()
		}
		if ms := m.GetSr().GetCores(); ms >= max.MaxCores {
			max.MaxCores = ms
			max.MaxCoresAt = m.GetTs()
		}
	}
	return max, nil
}
