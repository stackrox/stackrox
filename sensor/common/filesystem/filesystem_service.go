package filesystem

import (
	"context"
	"io"
	// "strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	// v1 "github.com/stackrox/rox/generated/api/v1"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	// "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	// "github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"google.golang.org/grpc"
)

const maxBufferSize = 10000

var (
	log = logging.LoggerForModule()
)

type Service interface {
	pkgGRPC.APIService
	sensorAPI.FileActivityServiceServer
}

type serviceImpl struct {
	unimplemented.Receiver
	sensorAPI.UnimplementedFileActivityServiceServer

	name string

	fsPipeline *Pipeline

	writer           io.Writer
	authFuncOverride func(context.Context, string) (context.Context, error)
}

func (s *serviceImpl) Name() string {
	return "filesystem.serviceImpl"
}

func (s *serviceImpl) Start() error {
	return nil
}

func (s *serviceImpl) Stop() {

}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensorAPI.RegisterFileActivityServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// There is no grpc gateway handler for signal service
	return nil
}

func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	err := idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
	return ctx, errors.Wrap(err, "collector authorization")
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return s.authFuncOverride(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(stream sensorAPI.FileActivityService_CommunicateServer) error {
	return s.receiveMessages(stream)
}

func (s *serviceImpl) receiveMessages(stream sensorAPI.FileActivityService_CommunicateServer) error {
	log.Info("Starting file system stream server")
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Error("error dequeueing file system activity event: ", err)
			return errors.Wrap(err, "receiving file system activity message")
		}

		log.Info("Got activity: ", msg)
		s.fsPipeline.Process(msg)
	}
}
