package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
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
	"github.com/stackrox/rox/pkg/safe"
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

	sensorHello, sensorSupportsHello, err := receiveSensorHello(server)
	if err != nil {
		return err
	}

	// Fetch the cluster metadata, then process the stream.
	cluster, err := s.getClusterForConnection(sensorHello, svc)
	if err != nil {
		return err
	}

	if sensorSupportsHello {
		// Let's be polite and respond with a greeting from our side.
		centralHello := &central.CentralHello{
			ClusterId: cluster.GetId(),
		}

		if err := safe.RunE(func() error {
			certBundle, err := clusters.IssueSecuredClusterCertificates(cluster, nil)
			if err != nil {
				return errors.Wrapf(err, "issuing a certificate bundle for cluster %s", cluster.GetName())
			}
			centralHello.CertBundle = certBundle.FileMap()
			return nil
		}); err != nil {
			log.Errorf("Could not include certificate bundle in sensor hello message: %s", err)
		}

		if err := server.Send(&central.MsgToSensor{Msg: &central.MsgToSensor_Hello{Hello: centralHello}}); err != nil {
			return errors.Wrap(err, "sending CentralHello message to sensor")
		}
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

	return s.manager.HandleConnection(server.Context(), sensorHello, cluster, eventPipeline, server)
}

func (s *serviceImpl) getClusterForConnection(sensorHello *central.SensorHello, serviceID *storage.ServiceIdentity) (*storage.Cluster, error) {
	var helmConfigInit *central.HelmManagedConfigInit
	if features.SensorInstallationExperience.Enabled() {
		helmConfigInit = sensorHello.GetHelmManagedConfigInit()
	}

	clusterIDFromCert := serviceID.GetId()
	if features.SensorInstallationExperience.Enabled() {
		if helmConfigInit == nil && clusterIDFromCert == centralsensor.InitCertClusterID {
			return nil, status.Error(codes.InvalidArgument, "sensor using cluster init certificate must be helm-managed")
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

	cluster, err := s.clusters.LookupOrCreateClusterFromConfig(clusterDSSAC, clusterID, sensorHello)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not fetch cluster for sensor: %v", err)
	}

	return cluster, nil
}

func receiveSensorHello(server central.SensorService_CommunicateServer) (*central.SensorHello, bool, error) {
	incomingMD := metautils.ExtractIncoming(server.Context())
	outMD := metautils.NiceMD{}

	sensorSupportsHello := incomingMD.Get(centralsensor.SensorHelloMetadataKey) == "true"
	if sensorSupportsHello {
		outMD.Set(centralsensor.SensorHelloMetadataKey, "true")
	}

	if err := server.SendHeader(metadata.MD(outMD)); err != nil {
		return nil, false, errors.Wrap(err, "sending header metadata")
	}

	if !sensorSupportsHello {
		sensorHello, err := centralsensor.DeriveSensorHelloFromIncomingMetadata(incomingMD)
		if err != nil {
			log.Warnf("Failed to completely derive SensorHello information from header metadata: %s", err)
		}

		return sensorHello, false, nil
	}

	firstMsg, err := server.Recv()
	if err != nil {
		return nil, false, errors.Wrap(err, "receiving first message")
	}
	sensorHello := firstMsg.GetHello()
	if sensorHello == nil {
		return nil, false, errors.Wrapf(err, "first message received is not a SensorHello message, but %T", firstMsg.GetMsg())
	}

	return sensorHello, true, nil
}
