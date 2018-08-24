package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sensor/metrics"
	"github.com/stackrox/rox/pkg/uuid"
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
	indicators chan *listeners.EventWrap // EventWrap is just a wrapper around Indicator
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSignalServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSensorEventServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushSignals(stream v1.SignalService_PushSignalsServer) error {
	_, err := authn.FromTLSContext(stream.Context())
	if err != nil {
		return err
	}
	s.receiveMessages(stream)
	return nil
}

func (s *serviceImpl) Indicators() <-chan *listeners.EventWrap {
	return s.indicators
}

func (s *serviceImpl) receiveMessages(stream v1.SignalService_PushSignalsServer) error {
	clientClusterID := env.ClusterID.Setting()

	for {
		signal, err := stream.Recv()
		if err != nil {
			log.Error("error dequeueing signal event: ", err)
			continue
		}

		if stream.Context().Err() != nil {
			log.Error(stream.Context().Err())
			continue
		}

		// Ignore the collector register request
		if signal.GetSignal() == nil {
			continue
		}

		// TODO: For testing! Remove once end-to-end data pipeline is complete
		log.Infof("Obtained signal: %+v", signal)

		indicator := &v1.Indicator{
			Id:     uuid.NewV4().String(),
			Signal: signal.GetSignal(),
		}

		// Log lag metrics from collector
		lag := time.Now().Sub(protoconv.ConvertTimestampToTimeOrNow(indicator.GetSignal().GetTime()))
		metrics.RegisterSignalToIndicatorCreateLag(clientClusterID, float64(lag.Nanoseconds()))

		wrappedEvent := &listeners.EventWrap{
			SensorEvent: &v1.SensorEvent{
				Id:        indicator.GetId(),
				ClusterId: clientClusterID,
				Resource: &v1.SensorEvent_Indicator{
					Indicator: indicator,
				},
			},
		}

		select {
		case s.indicators <- wrappedEvent:
		default:
			// TODO: We may want to consider popping stuff from the channel here so that we only retain the most recent events
			metrics.RegisterSensorIndicatorChannelFullCounter(clientClusterID)
		}

	}
}
