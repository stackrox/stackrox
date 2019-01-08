package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/sensor/service/streamer"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	manager streamer.Manager
}

// New creates a new Service using the given manager.
func New(manager streamer.Manager) Service {
	return &serviceImpl{
		manager: manager,
	}
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterSensorServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(server central.SensorService_CommunicateServer) error {
	// Get the source cluster's ID.
	identity := authn.IdentityFromContext(server.Context())
	if identity == nil {
		return authz.ErrNotAuthorized("only sensor may access this API")
	}
	svc := identity.Service()
	if svc == nil {
		return authz.ErrNotAuthorized("only sensor may access this API")
	}

	clusterID := svc.GetId()

	// Create a Streamer for the cluster. Throw error if it already exists.
	streamer, err := s.manager.CreateStreamer(clusterID)
	if err != nil {
		return fmt.Errorf("unable to open stream to cluster %s: %s", clusterID, err)
	}
	streamer.Start(server)
	streamer.WaitUntilFinished()

	if removeErr := s.manager.RemoveStreamer(clusterID, streamer); removeErr != nil {
		log.Errorf("Could not remove sensor connection for cluster %s: %v", clusterID, removeErr)
	}
	return err
}
