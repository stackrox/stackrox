package collector

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/metrics"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

type Option func(*serviceImpl)

// WithAuthFuncOverride sets the AuthFuncOverride.
func WithAuthFuncOverride(overrideFn func(context.Context, string) (context.Context, error)) Option {
	return func(srv *serviceImpl) {
		srv.authFuncOverride = overrideFn
	}
}

type serviceImpl struct {
	sensor.UnimplementedCollectorServiceServer

	processQueue chan *sensor.ProcessSignal

	authFuncOverride func(context.Context, string) (context.Context, error)
}

func newService(opts ...Option) Service {
	s := &serviceImpl{
		processQueue:     make(chan *sensor.ProcessSignal, env.CollectorIServiceChannelBufferSize.IntegerSetting()),
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(s)
	}
	return s
}

func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return s.authFuncOverride(ctx, fullMethodName)
}

func (s *serviceImpl) PushProcesses(server sensor.CollectorService_PushProcessesServer) error {
	for {
		select {
		case <-server.Context().Done():
			return nil
		default:
			if err := s.pushProcesses(server); err != nil {
				log.Error("error dequeueing collector message, event: ", err)
				return err
			}
		}
	}
}

func (s *serviceImpl) pushProcesses(server sensor.CollectorService_PushProcessesServer) error {
	msg, err := server.Recv()
	if err != nil {
		return err
	}

	metrics.CollectorChannelInc()
	select {
	case s.processQueue <- msg:
		metrics.IncrementTotalProcessesAddedCounter()
	case <-server.Context().Done():
		return nil
	default:
		metrics.IncrementTotalProcessesDroppedCounter()
	}
	return nil
}

func (s *serviceImpl) PushNetworkConnectionInfo(server sensor.CollectorService_PushNetworkConnectionInfoServer) error {
	// This method is not implemented on the collector side yet.
	return errors.New("Unimplemented PushNetworkConnectionInfo method")
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	sensor.RegisterCollectorServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

func (s *serviceImpl) GetMessagesC() <-chan *sensor.ProcessSignal {
	return s.processQueue
}
