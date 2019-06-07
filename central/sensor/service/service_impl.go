package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	manager  connection.Manager
	pf       pipeline.Factory
	clusters clusterDataStore.DataStore
}

// New creates a new Service using the given manager.
func New(manager connection.Manager, pf pipeline.Factory, clusters clusterDataStore.DataStore) Service {
	return &serviceImpl{
		manager:  manager,
		pf:       pf,
		clusters: clusters,
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
	if svc == nil || svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return authz.ErrNotAuthorized("only sensor may access this API")
	}

	// Fetch the cluster metadata, then process the stream.
	clusterID := svc.GetId()
	_, exists, err := s.clusters.GetCluster(server.Context(), clusterID)
	if err != nil {
		return status.Errorf(codes.Internal, "couldn't look-up cluster %q: %v", clusterID, err)
	}
	if !exists {
		return status.Errorf(codes.NotFound, "cluster %q not found in DB; it was possibly deleted", clusterID)
	}
	return s.manager.HandleConnection(server.Context(), clusterID, s.pf, server, s.clusters)
}
