package collector

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

type Option func(*serviceImpl)

type serviceImpl struct {
	sensor.UnimplementedCollectorServiceServer

	queue chan *sensor.ProcessSignal
}

func newService(queue chan *sensor.ProcessSignal) Service {
	return &serviceImpl{
		queue: queue,
	}
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(server sensor.CollectorService_CommunicateServer) error {
	for {
		msg, err := server.Recv()
		if err != nil {
			log.Error("error dequeueing collector message, event: ", err)
			return err
		}

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

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	sensor.RegisterCollectorServiceServer(server, s)
}
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}
