package service

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	v1 "github.com/stackrox/rox/generated/api/v1"
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
			"/v1.AdministrationUsageService/GetCurrentSecuredUnitsUsage",
			"/v1.AdministrationUsageService/GetMaxSecuredUnitsUsage",
		}})
)

type serviceImpl struct {
	v1.UnimplementedAdministrationUsageServiceServer

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
	v1.RegisterAdministrationUsageServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAdministrationUsageServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetCurrentSecuredUnitsUsage(ctx context.Context, _ *v1.Empty) (*v1.SecuredUnitsUsageResponse, error) {
	m, err := s.datastore.GetCurrentUsage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get current administration usage")
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
			return nil, errox.InvalidArgs.New("invalid value in from parameter").CausedBy(err)
		}
	}
	if req.GetTo() != nil {
		if to, err = types.TimestampFromProto(req.GetTo()); err != nil {
			return nil, errox.InvalidArgs.New("invalid value in to parameter").CausedBy(err)
		}
	}
	if !from.Before(to) {
		return nil, errox.InvalidArgs.New("bad combination of from and to parameters")
	}

	maxNumNodes, err := s.datastore.GetMaxNumNodes(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get maximum nodes usage")
	}
	max.MaxNodes = maxNumNodes.GetNumNodes()
	max.MaxNodesAt = maxNumNodes.GetTimestamp()

	maxNumCPUUnits, err := s.datastore.GetMaxNumCPUUnits(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get maximum CPU usage")
	}
	max.MaxCpuUnits = maxNumCPUUnits.GetNumCpuUnits()
	max.MaxCpuUnitsAt = maxNumCPUUnits.GetTimestamp()

	return max, nil
}
