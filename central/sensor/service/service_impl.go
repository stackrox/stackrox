package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

var (
	clusterDSSAC = sac.WithAllAccess(context.Background())
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
	cluster, err := s.getClusterForConnection(server, svc)
	if err != nil {
		return err
	}

	if expiry := identity.Expiry(); !expiry.IsZero() {
		converted, err := types.TimestampProto(expiry)
		if err != nil {
			log.Warnf("Failed to convert expiry of sensor cert (%v) from cluster %s to proto: %v", expiry, cluster.GetId(), err)
		} else {
			if err := s.clusters.UpdateClusterCertExpiryStatus(clusterDSSAC, cluster.GetId(), &storage.ClusterCertExpiryStatus{SensorCertExpiry: converted}); err != nil {
				log.Warnf("Failed to update cluster expiry status for cluster %s: %v", cluster.GetId(), err)
			}
		}
	}

	// Generate a pipeline for the cluster to use.
	eventPipeline, err := s.pf.PipelineForCluster(server.Context(), cluster.GetId())
	if err != nil {
		return status.Errorf(codes.Internal, "unable to generate a pipeline for cluster %q", cluster.GetId())
	}

	log.Infof("Cluster %s (%s) has successfully connected to Central", cluster.GetName(), cluster.GetId())

	return s.manager.HandleConnection(server.Context(), cluster, eventPipeline, server)
}

func (s *serviceImpl) getClusterForConnection(server central.SensorService_CommunicateServer, serviceID *storage.ServiceIdentity) (*storage.Cluster, error) {
	incomingMD := metautils.ExtractIncoming(server.Context())

	helmManaged := false
	if features.SensorInstallationExperience.Enabled() {
		helmManaged = incomingMD.Get(centralsensor.HelmManagedClusterMetadataKey) == "true"
	}

	clusterIDFromCert := serviceID.GetId()
	outMD := metautils.NiceMD{}
	if features.SensorInstallationExperience.Enabled() {
		if helmManaged {
			outMD.Set(centralsensor.HelmManagedClusterMetadataKey, "true")
		} else if clusterIDFromCert == centralsensor.InitCertClusterID {
			return nil, status.Error(codes.InvalidArgument, "sensor using cluster init certificate must be helm-managed")
		}
	}

	if err := server.SendHeader(metadata.MD(outMD)); err != nil {
		return nil, status.Error(codes.Internal, "failed to send header metadata to client")
	}

	var helmConfigInit *central.HelmManagedConfigInit
	if helmManaged {
		// For Helm-managed clusters, we expect to receive the cluster configuration as the first message from sensor.
		msg, err := server.Recv()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "receiving helm config init message: %v", err)
		}

		helmConfigInit = msg.GetHelmManagedConfigInit()
		if helmConfigInit == nil {
			return nil, status.Errorf(codes.InvalidArgument, "expected helm config init message, got %T", msg.GetMsg())
		}
	}

	clusterID := helmConfigInit.GetClusterId()
	if clusterID != "" || clusterIDFromCert != centralsensor.InitCertClusterID {
		var err error
		clusterID, err = centralsensor.GetClusterID(clusterID, clusterIDFromCert)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "incompatible cluster IDs in config init and certificate: %v", err)
		}
	}

	cluster, err := s.clusters.LookupOrCreateClusterFromConfig(clusterDSSAC, clusterID, helmConfigInit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not fetch cluster for sensor: %v", err)
	}

	return cluster, nil
}
