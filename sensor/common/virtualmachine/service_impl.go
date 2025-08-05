package virtualmachine

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

type serviceImpl struct {
	sensor.UnimplementedVirtualMachineServiceServer
	component Component
}

var _ Service = (*serviceImpl)(nil)

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterVirtualMachineServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	if err := idcheck.CollectorOnly().Authorized(ctx, fullMethodName); err != nil {
		return ctx, errors.Wrapf(err, "virtual machine authorization for %q", fullMethodName)
	}
	return ctx, nil
}

func (s *serviceImpl) UpsertVirtualMachine(ctx context.Context, req *sensor.UpsertVirtualMachineRequest) (*sensor.UpsertVirtualMachineResponse, error) {
	if req.VirtualMachine == nil {
		return &sensor.UpsertVirtualMachineResponse{
			Success: false,
		}, errox.InvalidArgs.CausedBy("virtual machine in request cannot be nil")
	}

	log.Debugf("Upserting virtual machine: %s", req.VirtualMachine.GetId())
	if err := s.component.Send(ctx, req.GetVirtualMachine()); err != nil {
		return &sensor.UpsertVirtualMachineResponse{
			Success: false,
		}, errors.Wrap(err, "sending virtual machine to Central")
	}
	return &sensor.UpsertVirtualMachineResponse{
		Success: true,
	}, nil
}
