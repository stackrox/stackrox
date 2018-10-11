package service

import (
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/sensorevent/service/streamer"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Service is the struct that manages the SensorEvent API
type serviceImpl struct {
	streamManager streamer.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSensorEventServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSensorEventServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

// RecordEvent takes in a stream of deployment events and outputs a stream of alerts and enforcement actions.
func (s *serviceImpl) RecordEvent(stream v1.SensorEventService_RecordEventServer) error {
	// Get the source cluster's ID.
	identity := authn.IdentityFromContext(stream.Context())
	if identity == nil {
		return authz.ErrNotAuthorized("only sensor may access this API")
	}
	svc := identity.Service()
	if svc == nil {
		return authz.ErrNotAuthorized("only sensor may access this API")
	}

	clientClusterID := svc.GetId()

	// Create a stream for the cluster. Throw error if it already exists.
	sensorStreamer, err := s.streamManager.CreateStreamer(clientClusterID)
	if err != nil {
		return fmt.Errorf("unable to open stream to cluster %s: %s", clientClusterID, err)
	}

	// Start the sensor stream and wait until it is empty.
	sensorStreamer.Start(stream)
	sensorStreamer.WaitUntilFinished()

	if err := s.streamManager.RemoveStreamer(clientClusterID); err != nil {
		return fmt.Errorf("unable to close stream to cluster %s: %s", clientClusterID, err)
	}

	return nil
}
