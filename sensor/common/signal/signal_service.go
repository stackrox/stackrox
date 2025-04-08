package signal

import (
	"context"
	"errors"

	"github.com/pkg/errors"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/metrics"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Option function for the signal service.
type Option func(*serviceImpl)

// WithAuthFuncOverride sets the AuthFuncOverride.
func WithAuthFuncOverride(overrideFn func(context.Context, string) (context.Context, error)) Option {
	return func(srv *serviceImpl) {
		srv.authFuncOverride = overrideFn
	}
}

// Service is the interface that manages the SignalEvent API from the server side
type Service interface {
	pkgGRPC.APIService
	sensorAPI.SignalServiceServer

	GetMessagesC() <-chan *storage.ProcessSignal
}

type serviceImpl struct {
	sensorAPI.UnimplementedSignalServiceServer

	queue chan *storage.ProcessSignal

	authFuncOverride func(context.Context, string) (context.Context, error)
}

func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	err := idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
	return ctx, errors.Wrap(err, "collector authorization")
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensorAPI.RegisterSignalServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// There is no grpc gateway handler for signal service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return s.authFuncOverride(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushSignals(stream sensorAPI.SignalService_PushSignalsServer) error {
	return s.receiveMessages(stream)
}

func (s *serviceImpl) GetMessagesC() <-chan *storage.ProcessSignal {
	return s.queue
}

func (s *serviceImpl) receiveMessages(stream sensorAPI.SignalService_PushSignalsServer) error {
	for {
		signalStreamMsg, err := stream.Recv()
		if err != nil {
			log.Error("error dequeueing signalStreamMsg event: ", err)
			return errors.Wrap(err, "receiving signal stream message")
		}

		signal, err := unwrapSignal(signalStreamMsg)
		if err != nil {
			metrics.IncrementTotalProcessesDroppedCounter()
			return err
		}

		metrics.IncrementTotalProcessesReceivedCounter()
		s.queue <- signal
	}
}

func unwrapSignal(msg *sensorAPI.SignalStreamMessage) (*storage.ProcessSignal, error) {
	signal := msg.GetSignal()
	if signal == nil {
		return nil, errors.New("empty signal")
	}

	processSignal := signal.GetProcessSignal()
	if processSignal == nil {
		return nil, errors.New("empty process signal")
	}

	return processSignal, nil
}
