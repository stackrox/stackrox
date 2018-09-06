package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensor/metrics"
	"google.golang.org/grpc"
)

const maxBufferSize = 10000

var (
	log = logging.LoggerForModule()
)

// Service is the interface that manages the SignalEvent API from the server side
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	PushSignals(stream v1.SignalService_PushSignalsServer) error
	Indicators() <-chan *listeners.EventWrap
}

type serviceImpl struct {
	queue      chan *v1.Signal
	indicators chan *listeners.EventWrap // EventWrap is just a wrapper around ProcessIndicator
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSignalServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// There is no grpc gateway handler for signal service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
	//return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushSignals(stream v1.SignalService_PushSignalsServer) error {
	log.Info("PushSignals called")
	s.receiveMessages(stream)
	return nil
}

func (s *serviceImpl) Indicators() <-chan *listeners.EventWrap {
	return s.indicators
}

func (s *serviceImpl) receiveMessages(stream v1.SignalService_PushSignalsServer) error {
	log.Info("starting receiveMessages")
	for {
		signalStreamMsg, err := stream.Recv()
		if err != nil {
			log.Error("error dequeueing signalStreamMsg event: ", err)
			return err
		}

		if stream.Context().Err() != nil {
			log.Error(stream.Context().Err())
			continue
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
			s.processProcessSignal(signal)
		default:
			// Currently eat unhandled signals
			continue
		}
	}
}

func (s *serviceImpl) pushEventToChannel(eventWrap *listeners.EventWrap) {
	select {
	case s.indicators <- eventWrap:
	default:
		// TODO: We may want to consider popping stuff from the channel here so that we only retain the most recent events
		metrics.RegisterSensorIndicatorChannelFullCounter(env.ClusterID.Setting())
	}
}
