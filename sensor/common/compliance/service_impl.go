package compliance

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// BenchmarkResultsService is the struct that manages the benchmark results API
type serviceImpl struct {
	output chan *central.MsgFromSensor
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterComplianceServiceServer(grpcServer, s)
}

// RegisterServiceHandler implements the APIService interface, but the agent does not accept calls over the gRPC gateway
func (s *serviceImpl) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// TODO:: Provide credentials to the benchmark service and verify them here.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// Output returns the channel where the received messages are output.
func (s *serviceImpl) Output() <-chan *central.MsgFromSensor {
	return s.output
}

// PushComplianceReturn takes the compliance results and outputs them to the channel.
func (s *serviceImpl) PushComplianceReturn(ctx context.Context, request *compliance.ComplianceReturn) (*v1.Empty, error) {
	// Push a message to the output channel.
	s.output <- returnAsMessage(request)
	return &v1.Empty{}, nil
}

// Helper function to make formatting values easier.
func returnAsMessage(cr *compliance.ComplianceReturn) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_ComplianceReturn{
					ComplianceReturn: cr,
				},
			},
		},
	}
}
