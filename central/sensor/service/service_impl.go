package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/sensor/manager"
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
	manager manager.SensorManager
}

// New creates a new Service using the given manager.
func New(manager manager.SensorManager) Service {
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

	// Create a stream for the cluster. Throw error if it already exists.
	sensorConn, err := s.manager.CreateConnection(clusterID)
	if err != nil {
		return fmt.Errorf("unable to open stream to cluster %s: %s", clusterID, err)
	}

	err = sensorConn.Communicate(server)
	if removeErr := s.manager.RemoveConnection(clusterID, sensorConn); removeErr != nil {
		log.Errorf("Could not remove sensor connection for cluster %s: %v", clusterID, removeErr)
	}
	return err
}
