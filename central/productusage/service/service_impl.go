package service

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.ProductUsageService/GetCurrentSecuredUnitsUsage",
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
	return v1.RegisterProductUsageServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetCurrentSecuredUnitsUsage(ctx context.Context, _ *v1.Empty) (*v1.SecuredUnitsUsageResponse, error) {
	m, err := s.datastore.GetCurrentUsage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get current product usage")
	}
	return &v1.SecuredUnitsUsageResponse{
		NumNodes:    m.GetNumNodes(),
		NumCpuUnits: m.GetNumCpuUnits(),
	}, nil
}

func (s *serviceImpl) GetMaxSecuredUnitsUsage(ctx context.Context, req *v1.TimeRange) (*v1.MaxSecuredUnitsUsageResponse, error) {
	max := &v1.MaxSecuredUnitsUsageResponse{}
	var from time.Time
	to := time.Now()
	var err error
	if req.GetFrom() != nil {
		if from, err = types.TimestampFromProto(req.GetFrom()); err != nil {
			return nil, errox.InvalidArgs.New("invalid value in from parameter")
		}
	}
	if req.GetTo() != nil {
		if to, err = types.TimestampFromProto(req.GetTo()); err != nil {
			return nil, errox.InvalidArgs.New("invalid value in to parameter")
		}
	}
	if !from.Before(to) {
		return nil, errox.InvalidArgs.New("bad combination of from and to parameters")
	}
	if err := s.datastore.Walk(ctx, from, to,
		func(metrics *storage.SecuredUnits) error {
			if nodes := metrics.GetNumNodes(); nodes >= max.MaxNodes {
				max.MaxNodes = nodes
				max.MaxNodesAt = metrics.GetTimestamp()
			}
			if cpus := metrics.GetNumCpuUnits(); cpus >= max.MaxCpuUnits {
				max.MaxCpuUnits = cpus
				max.MaxCpuUnitsAt = metrics.GetTimestamp()
			}
			return nil
		}); err != nil {
		return nil, errors.Wrap(err, "cannot get product usage")
	}
	return max, nil
}
