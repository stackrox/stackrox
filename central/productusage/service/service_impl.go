package usage

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.ProductUsageService/GetCurrentProductUsage",
			"/v1.ProductUsageService/GetMaxSecuredUnitsUsage",
		}})
)

type serviceImpl struct {
	v1.UnimplementedProductUsageServiceServer

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
	v1.RegisterProductUsageServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return errors.Wrap(v1.RegisterProductUsageServiceHandler(ctx, mux, conn), "failed to register the usage service handler")
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, errors.Wrapf(authorizer.Authorized(ctx, fullMethodName), "failed to authorize a call to %s", fullMethodName)
}

func (s *serviceImpl) GetCurrentProductUsage(ctx context.Context, _ *v1.Empty) (*v1.CurrentProductUsageResponse, error) {
	m, err := s.datastore.GetCurrentUsage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "datastore failed to get current usage metrics")
	}
	return &v1.CurrentProductUsageResponse{
		Timestamp: m.GetTimestamp(),
		SecuredUnits: &v1.SecuredUnits{
			NumNodes:    m.GetNumNodes(),
			NumCpuUnits: m.GetNumCpuUnits(),
		}}, nil
}

func (s *serviceImpl) GetMaxSecuredUnitsUsage(ctx context.Context, req *v1.TimeRange) (*v1.MaxSecuredUnitsUsageResponse, error) {
	metrics, err := s.datastore.Get(ctx, req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, errors.Wrap(err, "cannot get usage")
	}
	max := &v1.MaxSecuredUnitsUsageResponse{}
	for m := range metrics {
		if n := m.GetNumNodes(); n >= max.MaxNodes {
			max.MaxNodes = n
			max.MaxNodesAt = m.GetTimestamp()
		}
		if ms := m.GetNumCpuUnits(); ms >= max.MaxCpuUnits {
			max.MaxCpuUnits = ms
			max.MaxCpuUnitsAt = m.GetTimestamp()
		}
	}
	return max, nil
}
