package virtualmachine

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
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
	Notify(e common.SensorComponentEvent)
	Start() error
	Stop(err error) // TODO: get rid of err argument as it always seems to be effectively nil.
	Capabilities() []centralsensor.SensorCapability
	ProcessMessage(msg *central.MsgToSensor) error
	ResponsesC() <-chan *message.ExpiringMessage
}

// NewService returns the VirtualMachineServiceServer API for Sensor to use.
func NewService() Service {
	return &serviceImpl{
		stopper:        concurrency.NewStopper(),
		fromDataSource: make(chan *storage.VirtualMachine, 10),
	}
}

type serviceImpl struct {
	sensor.UnimplementedVirtualMachineServiceServer
	toCentral      chan *message.ExpiringMessage
	stopper        concurrency.Stopper
	fromDataSource chan *storage.VirtualMachine
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
	log.Infof("vm: %v", req.VirtualMachine.Id)
	if s.toCentral == nil {
		return &sensor.UpsertVirtualMachineResponse{
			Success: false,
		}, errors.New("Connection to Central is not ready")
	}
	if req.VirtualMachine != nil {
		log.Infof("Upserting virtual machine: %s", req.VirtualMachine.Id)
		s.fromDataSource <- req.VirtualMachine
		return &sensor.UpsertVirtualMachineResponse{
			Success: true,
		}, nil
	} else {
		log.Info("Virtual machine is nil")
	}
	return &sensor.UpsertVirtualMachineResponse{
		Success: false,
	}, nil
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
}

func (s *serviceImpl) Start() error {
	log.Infof("Starting virtual machine component!")
	ch2Central := make(chan *message.ExpiringMessage)
	go func() {
		defer func() {
			s.stopper.Flow().ReportStopped()
			close(ch2Central)
		}()
		for {
			select {
			case <-s.stopper.Flow().StopRequested():
				return
			case vm := <-s.fromDataSource:
				log.Info("Relaying vm event to s.toCentral")
				s.toCentral <- message.New(&central.MsgFromSensor{
					Msg: &central.MsgFromSensor_Event{
						Event: &central.SensorEvent{
							Id: vm.Id,
							// ResourceAction_UNSET_ACTION_RESOURCE is the only one supported by Central 4.6 and older.
							// This can be changed to CREATE or UPDATE for Sensor 4.8 or when Central 4.6 is out of support.
							Action: central.ResourceAction_UNSET_ACTION_RESOURCE,
							Resource: &central.SensorEvent_VirtualMachine{
								VirtualMachine: vm,
							},
						},
					},
				})
			}
		}
	}()
	s.toCentral = ch2Central
	return nil
}

func (s *serviceImpl) Stop(err error) {
	if !s.stopper.Client().Stopped().IsDone() {
		defer utils.IgnoreError(s.stopper.Client().Stopped().Wait)
	}
	s.stopper.Client().Stop()
}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (s *serviceImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return s.toCentral
}
