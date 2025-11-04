package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
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
		stoppers:     set.NewSet[concurrency.Stopper](),
	}

	return srv
}

type serviceImpl struct {
	sensor.UnimplementedFileActivityServiceServer
	pipeline     *pipeline.Pipeline
	activityChan chan *sensorAPI.FileActivity
	stoppers     set.Set[concurrency.Stopper]
}

func (s *serviceImpl) Stop() {
	// Stop all active connections
	for _, stopper := range s.stoppers.AsSlice() {
		stopper.Client().Stop() // Signal the receiveMessages that it needs to stop
	}
	// Wait for all connections to stop
	for _, stopper := range s.stoppers.AsSlice() {
		<-stopper.Client().Stopped().Done() // Wait for receiveMessages to stop
	}
	// Close the channel first to signal no more messages
	close(s.activityChan)
	// Wait for the pipeline to finish processing
	s.pipeline.Stop()
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
	// Create a stopper for this agent connection
	stopper := concurrency.NewStopper()
	s.stoppers.Add(stopper)
	defer s.stoppers.Remove(stopper)

	return s.receiveMessages(stream, stopper)
}

func (s *serviceImpl) receiveMessages(stream sensor.FileActivityService_CommunicateServer, stopper concurrency.Stopper) error {
	log.Info("Starting file system stream server")
	defer stopper.Flow().ReportStopped() // Signal the function has stopped
	for {
		msg, err := stream.Recv()
		if err != nil {
			return errors.Wrap(err, "receiving file system activity message")
		}

		log.Debug("Got file activity: ", msg)
		select {
		case <-stopper.Flow().StopRequested(): // Stop the function
			return nil
		case s.activityChan <- msg:
		}
	}
}
