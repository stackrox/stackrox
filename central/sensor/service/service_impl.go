package service

import (
	"context"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	installationStore "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/central/metrics/telemetry"
	"github.com/stackrox/rox/central/securedclustercertgen"
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
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	protoconv "github.com/stackrox/rox/pkg/protoconv/certs"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/safe"
	"github.com/stackrox/rox/pkg/set"
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
	sensorsOrRegistrants := or.Or(idcheck.SensorsOnly(), idcheck.SensorRegistrantsOnly())
	return ctx, sensorsOrRegistrants.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) Communicate(server central.SensorService_CommunicateServer) error {
	// Get the source cluster's ID.
	identity, err := authn.IdentityFromContext(server.Context())
	if err != nil {
		return err
	}

	svc := identity.Service()
	if svc == nil {
		return errox.NotAuthorized.CausedBy("missing service identity for this API")
	}
	svcType := svc.GetType()
	if !(svcType == storage.ServiceType_SENSOR_SERVICE || svcType == storage.ServiceType_REGISTRANT_SERVICE) {
		return errox.NotAuthorized.CausedByf("only sensor may access this API, unexpected client identity %q", svcType)
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
	clusterID := cluster.GetId()

	// Disallow secured cluster impersonation using a leaked CRS certificate.
	// Note: New clusters which have never been properly connected to central have their HealthStatus.LastContact empty.
	// If, for whatever reason, a cluster needs to re-retrieve CRS-issued service certificates, the cluster can do so
	// as long as that cluster has never successfully connected to central with proper service certificates
	// (not ServiceType_REGISTRANT_SERVICE).
	if svcType == storage.ServiceType_REGISTRANT_SERVICE && cluster.GetHealthStatus().GetLastContact() != nil {
		log.Errorf("It is forbidden to connect with a Cluster Registration Certificate as already-existing cluster %q.", cluster.GetName())
		return errox.NotAuthorized.CausedByf("forbidden to use a Cluster Registration Certificate for already-existing cluster %q", cluster.GetName())
	}

	// Generate a pipeline for the cluster to use.
	eventPipeline, err := s.pf.PipelineForCluster(server.Context(), cluster.GetId())
	if err != nil {
		return errors.Errorf("unable to generate a pipeline for cluster %q", cluster.GetId())
	}

	if sensorSupportsHello {
		installInfo, err := telemetry.FetchInstallInfo(context.Background(), s.installation)
		utils.Should(err)

		capabilities := sliceutils.StringSlice(eventPipeline.Capabilities()...)
		capabilities = append(capabilities, centralsensor.SecuredClusterCertificatesReissue)
		if features.SensorReconciliationOnReconnect.Enabled() {
			capabilities = append(capabilities, centralsensor.SendDeduperStateOnReconnect)
		}
		if features.ComplianceEnhancements.Enabled() {
			capabilities = append(capabilities, centralsensor.ComplianceV2Integrations)
		}
		if features.ComplianceRemediationV2.Enabled() {
			capabilities = append(capabilities, centralsensor.ComplianceV2Remediations)
		}
		if features.ScannerV4.Enabled() {
			capabilities = append(capabilities, centralsensor.ScannerV4Supported)
		}
		if features.ClusterRegistrationSecrets.Enabled() {
			capabilities = append(capabilities, centralsensor.ClusterRegistrationSecretSupported)
		}

		preferences := s.manager.GetConnectionPreference(clusterID)

		// Let's be polite and respond with a greeting from our side.
		centralHello := &central.CentralHello{
			ClusterId:        clusterID,
			ManagedCentral:   env.ManagedCentral.BooleanSetting(),
			CentralId:        installInfo.GetId(),
			Capabilities:     capabilities,
			SendDeduperState: preferences.SendDeduperState,
		}

		if err := safe.RunE(func() error {
			sensorNamespace := sensorHello.GetDeploymentIdentification().GetAppNamespace()
			certificateSet, err := securedclustercertgen.IssueSecuredClusterCerts(
				sensorNamespace, clusterID, isCARotationSupported(sensorHello))
			if err != nil {
				return errors.Wrapf(err, "issuing a certificate bundle for cluster %s", cluster.GetName())
			}
			centralHello.CertBundle, err = protoconv.ConvertTypedServiceCertificateSetToFileMap(certificateSet)
			if err != nil {
				return errors.Wrap(err, "converting typed service certificate set to file map")
			}
			return nil
		}); err != nil {
			log.Errorf("Could not include certificate bundle in sensor hello message: %s", err)
		}

		if err := server.Send(&central.MsgToSensor{Msg: &central.MsgToSensor_Hello{Hello: centralHello}}); err != nil {
			return errors.Wrap(err, "sending CentralHello message to sensor")
		}
	}

	if svcType == storage.ServiceType_REGISTRANT_SERVICE {
		// Terminate connection which uses a CRS certificate at this point.
		return nil
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

func isCARotationSupported(sensorHello *central.SensorHello) bool {
	capabilities := sliceutils.FromStringSlice[centralsensor.SensorCapability](sensorHello.GetCapabilities()...)
	capSet := set.NewSet(capabilities...)
	return capSet.Contains(centralsensor.SensorCARotationSupported)
}

func getCertExpiryStatus(identity authn.Identity) (*storage.ClusterCertExpiryStatus, error) {
	notBefore, notAfter := identity.ValidityPeriod()

	if notAfter.IsZero() && notBefore.IsZero() {
		return nil, nil
	}

	expiryStatus := &storage.ClusterCertExpiryStatus{}

	var multiErr error

	if !notAfter.IsZero() {
		expiryTimestamp, err := protocompat.ConvertTimeToTimestampOrError(notAfter)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		} else {
			expiryStatus.SensorCertExpiry = expiryTimestamp
		}
	}
	if !notBefore.IsZero() {
		notBeforeTimestamp, err := protocompat.ConvertTimeToTimestampOrError(notBefore)
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
	outMD := metautils.MD{}

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
