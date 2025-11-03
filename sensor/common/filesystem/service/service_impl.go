package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/filesystem/pipeline"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// NewService creates a new streaming service with the fact agent. It should only be called once.
func NewService(pipeline *pipeline.Pipeline, activityChan chan *sensorAPI.FileActivity) Service {
	srv := &serviceImpl{
		pipeline:     pipeline,
		activityChan: activityChan,
	}

	return srv
}

type serviceImpl struct {
	sensor.UnimplementedFileActivityServiceServer
	pipeline     *pipeline.Pipeline
	activityChan chan *sensorAPI.FileActivity
}

func (s *serviceImpl) Stop() {
	close(s.activityChan)
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterFileActivityServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// There is no grpc gateway handler for fact
	return nil
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, errors.Wrapf(idcheck.CollectorOnly().Authorized(ctx, fullMethodName), "file activity authorization for  %q", fullMethodName)
}

func (s *serviceImpl) Communicate(stream sensor.FileActivityService_CommunicateServer) error {
	return s.receiveMessages(stream)
}

func (s *serviceImpl) receiveMessages(stream sensor.FileActivityService_CommunicateServer) error {
	log.Info("Starting file system stream server")
	for {
		msg, err := stream.Recv()
		if err != nil {
			return errors.Wrap(err, "receiving file system activity message")
		}

		log.Debug("Got file activity: ", msg)
		s.activityChan <- msg
	}
}
