package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	installationStore "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/central/metrics/telemetry"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/safe"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	log = logging.LoggerForModule()
)

var (
	clusterDSSAC = sac.WithAllAccess(context.Background())
)

type serviceImpl struct {
	central.UnimplementedSensorServiceServer

	manager      connection.Manager
	pf           pipeline.Factory
	clusters     clusterDataStore.DataStore
	installation installationStore.Store
}

// New creates a new Service using the given manager.
func New(manager connection.Manager, pf pipeline.Factory, clusters clusterDataStore.DataStore, installation installationStore.Store) Service {
	return &serviceImpl{
		manager:      manager,
		pf:           pf,
		clusters:     clusters,
		installation: installation,
	}
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	central.RegisterSensorServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(server central.SensorService_CommunicateServer) error {
	// Get the source cluster's ID.
	identity, err := authn.IdentityFromContext(server.Context())
	if err != nil {
		return err
	}

	svc := identity.Service()
	if svc == nil || svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return errox.NotAuthorized.CausedBy("only sensor may access this API")
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

	// Generate a pipeline for the cluster to use.
	eventPipeline, err := s.pf.PipelineForCluster(server.Context(), cluster.GetId())
	if err != nil {
		return errors.Errorf("unable to generate a pipeline for cluster %q", cluster.GetId())
	}

	if sensorSupportsHello {
		installInfo, err := telemetry.FetchInstallInfo(context.Background(), s.installation)
		utils.Should(err)

		// Check if there is a deduper state available for this cluster. Otherwise, central should
		// request Sensor to not wait for any state. This is to avoid the case where an old backup
		// is restored which causes the deduper table to be empty but deployments are still in the
		// deployments table. This will cause sensor to not send the deletes for deployments that
		// are not transmitted in the deduper state message.
		deduperForCluster := hashManager.Singleton().GetDeduper(context.Background(), cluster.GetId())

		capabilities := sliceutils.StringSlice(eventPipeline.Capabilities()...)
		if features.SensorReconciliationOnReconnect.Enabled() && len(deduperForCluster.GetSuccessfulHashes()) > 0 {
			capabilities = append(capabilities, centralsensor.SensorReconciliationOnReconnect)
		}

		// Let's be polite and respond with a greeting from our side.
		centralHello := &central.CentralHello{
			ClusterId:      cluster.GetId(),
			ManagedCentral: env.ManagedCentral.BooleanSetting(),
			CentralId:      installInfo.GetId(),
			Capabilities:   capabilities,
		}

		if err := safe.RunE(func() error {
			certBundle, err := clusters.IssueSecuredClusterCertificates(cluster, sensorHello.GetDeploymentIdentification().GetAppNamespace(), nil)
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

	if expiryStatus, err := getCertExpiryStatus(identity); err != nil {
		notBefore, notAfter := identity.ValidityPeriod()
		log.Warnf("Failed to convert expiry status of sensor cert (NotBefore: %v, Expiry: %v) from cluster %s to proto: %v",
			notBefore, notAfter, cluster.GetId(), err)
	} else if expiryStatus != nil {
		if err := s.clusters.UpdateClusterCertExpiryStatus(clusterDSSAC, cluster.GetId(), expiryStatus); err != nil {
			log.Warnf("Failed to update cluster expiry status for cluster %s: %v", cluster.GetId(), err)
		}
	}

	log.Infof("Cluster %s (%s) has successfully connected to Central", cluster.GetName(), cluster.GetId())

	return s.manager.HandleConnection(server.Context(), sensorHello, cluster, eventPipeline, server)
}

func getCertExpiryStatus(identity authn.Identity) (*storage.ClusterCertExpiryStatus, error) {
	notBefore, notAfter := identity.ValidityPeriod()

	if notAfter.IsZero() && notBefore.IsZero() {
		return nil, nil
	}

	expiryStatus := &storage.ClusterCertExpiryStatus{}

	var multiErr error

	if !notAfter.IsZero() {
		expiryTimestamp, err := types.TimestampProto(notAfter)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		} else {
			expiryStatus.SensorCertExpiry = expiryTimestamp
		}
	}
	if !notBefore.IsZero() {
		notBeforeTimestamp, err := types.TimestampProto(notBefore)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		} else {
			expiryStatus.SensorCertNotBefore = notBeforeTimestamp
		}
	}
	return expiryStatus, multiErr
}

func (s *serviceImpl) getClusterForConnection(sensorHello *central.SensorHello, serviceID *storage.ServiceIdentity) (*storage.Cluster, error) {
	helmConfigInit := sensorHello.GetHelmManagedConfigInit()

	clusterIDFromCert := serviceID.GetId()
	if helmConfigInit == nil && centralsensor.IsInitCertClusterID(clusterIDFromCert) {
		return nil, errors.Wrap(errox.InvalidArgs, "sensor using cluster init certificate must transmit a helm-managed configuration")
	}

	clusterID := helmConfigInit.GetClusterId()
	if clusterID != "" || !centralsensor.IsInitCertClusterID(clusterIDFromCert) {
		var err error
		clusterID, err = centralsensor.GetClusterID(clusterID, clusterIDFromCert)
		if err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "incompatible cluster IDs in config init and certificate: %v", err)
		}
	}

	cluster, err := s.clusters.LookupOrCreateClusterFromConfig(clusterDSSAC, clusterID, serviceID.InitBundleId, sensorHello)
	if err != nil {
		return nil, errors.Errorf("could not fetch cluster for sensor: %v", err)
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
