package virtualmachine

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Service is an interface provides functionality to get deployments from Sensor.
type Service interface {
	grpcPkg.APIService
	sensor.VirtualMachineServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// NewService returns the VirtualMachineServiceServer API for Sensor to use.
func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {
	sensor.UnimplementedVirtualMachineServiceServer
}

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
	if err := idcheck.AdmissionControlOnly().Authorized(ctx, fullMethodName); err != nil {
		return ctx, errors.Wrapf(err, "virtual machine authorization for %q", fullMethodName)
	}
	return ctx, nil
}

func (s *serviceImpl) UpsertVirtualMachine(ctx context.Context, req *sensor.UpsertVirtualMachineRequest) (*sensor.UpsertVirtualMachineResponse, error) {
	log.Infof("vm: %v", req.VirtualMachine)
	if req.VirtualMachine != nil {
		log.Infof("Upserting virtual machine: %s", req.VirtualMachine.Id)
	} else {
		log.Info("Virtual machine is nil")
	}
	return &sensor.UpsertVirtualMachineResponse{
		Success: true,
	}, nil
}
