package collector

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
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

	queue chan *sensor.ProcessSignal

	authFuncOverride func(context.Context, string) (context.Context, error)
}

func newService(opts ...Option) Service {
	s := &serviceImpl{
		queue:            make(chan *sensor.ProcessSignal, env.CollectorIServiceChannelBufferSize.IntegerSetting()),
		authFuncOverride: authFuncOverride,
	}

	for _, o := range opts {
		o(s)
	}
	return s
}

func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, errors.Wrap(idcheck.CollectorOnly().Authorized(ctx, fullMethodName), "Unauthorized access")
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return s.authFuncOverride(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(server sensor.CollectorService_CommunicateServer) error {
	for {
		select {
		case <-server.Context().Done():
			return nil
		default:
			if err := s.communicate(server); err != nil {
				log.Error("error dequeueing collector message, event: ", err)
				return err
			}
		}
	}
}

func (s *serviceImpl) communicate(server sensor.CollectorService_CommunicateServer) error {
	msg, err := server.Recv()
	if err != nil {
		return errors.Wrap(err, "Failed to receive message from collector")
	}

	metrics.CollectorChannelInc(msg)
	switch msg.GetMsg().(type) {
	case *sensor.MsgFromCollector_ProcessSignal:
		metrics.IncrementTotalProcessesAddedCounter()
		select {
		case s.queue <- msg.GetProcessSignal():
		case <-server.Context().Done():
			return nil
		default:
			metrics.IncrementTotalProcessesDroppedCounter()
		}
	case *sensor.MsgFromCollector_Register:
	case *sensor.MsgFromCollector_Info:
	default:
		log.Errorf("got unknown message from Collector: %T", msg.GetMsg())
	}
	return nil
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	sensor.RegisterCollectorServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

func (s *serviceImpl) GetMessagesC() <-chan *sensor.ProcessSignal {
	return s.queue
}
