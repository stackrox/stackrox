package service

import (
	"context"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// NewService creates a new streaming service with the fact agent. It should only be called once.
func NewService() Service {
	srv := &serviceImpl{
		authFuncOverride: authFuncOverride,
		writer:           nil,
	}

	return srv
}

func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	err := idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
	return ctx, errors.Wrapf(err, "file activity authorization for %q", fullMethodName)
}

type serviceImpl struct {
	sensor.UnimplementedFileActivityServiceServer

	authFuncOverride func(context.Context, string) (context.Context, error)
	writer           io.Writer
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

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return s.authFuncOverride(ctx, fullMethodName)
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
	}
}
