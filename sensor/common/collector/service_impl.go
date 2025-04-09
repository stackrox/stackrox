package collector

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensor/queue"
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
	queueSize := env.CollectorIServiceQueueSize
	s := &serviceImpl{
		queue: make(chan *sensor.ProcessSignal, queue.ScaleSizeOnNonDefault(queueSize)),
	}

	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(server sensor.CollectorService_CommunicateServer) error {
	defer close(s.queue)

	for {
		msg, err := server.Recv()
		if err != nil {
			log.Error("error dequeueing collector message, event: ", err)
			return err
		}

		select {
		case <-server.Context().Done():
			return nil
		default:
			metrics.CollectorChannelInc(msg)
			switch msg.GetMsg().(type) {
			case *sensor.MsgFromCollector_ProcessSignal:
				s.queue <- msg.GetProcessSignal()
			case *sensor.MsgFromCollector_Register:
				log.Infof("got register: %+v", msg.GetRegister())
			case *sensor.MsgFromCollector_Info:
				log.Infof("got network info: %+v", msg.GetInfo())
			default:
				log.Errorf("got unknown message type %T", msg.GetMsg())
			}
		}
	}
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
