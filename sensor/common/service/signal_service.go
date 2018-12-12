package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	sensor "github.com/stackrox/rox/sensor/common"
	"google.golang.org/grpc"
)

const maxBufferSize = 10000

var (
	log = logging.LoggerForModule()
)

// Service is the interface that manages the SignalEvent API from the server side
type Service interface {
	pkgGRPC.APIService
	sensorAPI.SignalServiceServer

	Indicators() <-chan *v1.SensorEvent
}

type serviceImpl struct {
	queue      chan *v1.Signal
	indicators chan *v1.SensorEvent

	processPipeline sensor.Pipeline
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensorAPI.RegisterSignalServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// There is no grpc gateway handler for signal service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushSignals(stream sensorAPI.SignalService_PushSignalsServer) error {
	s.receiveMessages(stream)
	return nil
}

func (s *serviceImpl) Indicators() <-chan *v1.SensorEvent {
	return s.indicators
}

func (s *serviceImpl) receiveMessages(stream sensorAPI.SignalService_PushSignalsServer) error {
	log.Info("starting receiveMessages")
	for {
		signalStreamMsg, err := stream.Recv()
		if err != nil {
			log.Error("error dequeueing signalStreamMsg event: ", err)
			return err
		}

		// Ignore the collector register request
		if signalStreamMsg.GetSignal() == nil {
			log.Error("Empty signalStreamMsg")
			continue
		}
		signal := signalStreamMsg.GetSignal()

		// todo(cgorman) we currently need to filter out network because they are not being processed
		switch signal.GetSignal().(type) {
		case *v1.Signal_ProcessSignal:
			processSignal := signal.GetProcessSignal()
			if processSignal == nil {
				log.Error("Empty process signal")
				continue
			}

			log.Debugf("Process Signal: %+v", processSignal)
			s.processPipeline.Process(processSignal)
		default:
			// Currently eat unhandled signals
			continue
		}
	}
}
