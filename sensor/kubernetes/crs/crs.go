package crs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/env"
	grpcUtil "github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/pods"
	protoconv "github.com/stackrox/rox/pkg/protoconv/certs"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralclient"
	sensorCommon "github.com/stackrox/rox/sensor/common/sensor"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/securedcluster"
	"github.com/stackrox/rox/sensor/kubernetes/helm"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"google.golang.org/grpc/metadata"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var (
	log                        = logging.LoggerForModule()
	clusterRegistrationTimeout = 2 * time.Minute
)

// EnsureServiceCertificatesPresent initiates the CRS-based cluster registration flow to retrieve the
// service certificates in case no services certificates are found and instead a CRS is present.
func EnsureServiceCertificatesPresent() error {
	log.Infof("Ensuring certificates for Secured Cluster services are present.")
	clientconn.SetUserAgent(fmt.Sprintf("%s Cluster Registration Helper", clientconn.Sensor))

	// Check if legacy sensor service certificate, e.g. created via init-bundle, exists.
	if legacySensorServiceCert := os.Getenv(crs.LegacySensorServiceCertEnvName); legacySensorServiceCert != "" {
		log.Infof("Legacy sensor service certificate available, skipping cluster registration using Cluster Registration Secret.")
		return nil
	}

	// Check if modern sensor service certificate, e.g. created by CRS-based cluster registration or
	// refreshed by periodic cert rotation, exists.
	if sensorServiceCert := os.Getenv(crs.SensorServiceCertEnvName); sensorServiceCert != "" {
		log.Infof("Sensor service certificate available, skipping CRS-based cluster registration.")
		return nil
	}

	log.Infof("No sensor service certificate found, initiating cluster registration using Cluster Registration Secret.")
	if err := registerCluster(); err != nil {
		return errors.Wrap(err, "registering secured cluster")
	}
	log.Info("Cluster registration complete.")

	return nil
}

// registerCluster implements the CRS-based registration flow for secured clusters.
// It
//   - retrieves the CRS secret
//   - unpacks the contained mTLS certificate+key pair for the REGISTRANT_SERVICE identity from the
//     CRS and stores it in /run/secrets/stackrox.io/certs (which must be a memory-backed
//     emptyDir) so that they will be picked up automatically for our mTLS authentication towards Central.
//   - connects to Central and sends a sensorHello message for this new secured cluster
//   - expects to receive a centralHello response containing newly issued service certificates+keys
//     which are then stored as Kubernetes secrets named `tls-cert-<service slug name>`.
func registerCluster() error {
	ctx, cancel := context.WithTimeout(context.Background(), clusterRegistrationTimeout)
	defer cancel()

	log.Infof("Trying to load Cluster Registration Secret.")
	crs, err := crs.Load()
	if err != nil {
		log.Errorf("failed to load CRS: %v", err)
		return errors.Wrap(err, "loading CRS")
	}

	err = temporarilyStoreRegistrantSecret(crs)
	if err != nil {
		return errors.Wrap(err, "preparing registration secret for mTLS authentication")
	}

	centralConnection, err := openCentralConnection()
	if err != nil {
		return errors.Wrap(err, "opening connection to Central")
	}

	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return errors.Wrap(err, "obtaining in-cluster Kubernetes config")
	}
	k8sClient := k8sutil.MustCreateK8sClient(config)

	centralHello, err := centralHandshake(ctx, k8sClient, centralConnection)
	if err != nil {
		return errors.Wrap(err, "handshake with central")
	}

	// Store certificates+keys contained in Central's centralHello response as
	// Kubernetes secrets named `tls-cert-<service slug name>`.
	err = persistCertificates(ctx, centralHello.GetCertBundle(), k8sClient)
	if err != nil {
		return errors.Wrap(err, "persisting certificates")
	}

	return nil
}

// temporarilyStoreRegistrantSecret extracts the REGISTRANT_SERVICE certificate+key pair from the CRS
// and stores it alongside the CA certificate in the temporary storage /run/secrets/stackrox.io/certs.
func temporarilyStoreRegistrantSecret(crs *crs.CRS) error {
	// Extract (first) CA certificate from the CRS.
	var caCert string
	if len(crs.CAs) > 0 {
		caCert = crs.CAs[0]
	}
	if caCert == "" {
		return errors.New("malformed Cluster Registration Secret (missing CA certificate)")
	}

	for fileName, content := range map[string]string{
		"ca.pem":   caCert,
		"cert.pem": crs.Cert,
		"key.pem":  crs.Key,
	} {
		filePath := filepath.Join(mtls.CertsPrefix, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			return errors.Wrapf(err, "writing mTLS material to file %q", filePath)
		}
		log.Infof("Successfully wrote file %s.", filePath)
	}

	return nil
}

// dummyClusterIDHandler is a dummy clusterid.Handler
// This is needed because SetCentralConnectionWithRetries expects it,
// but it is not really used in the CRD container
type dummyClusterIDHandler struct{}

func (d *dummyClusterIDHandler) GetNoWait() string {
	return ""
}

func (d *dummyClusterIDHandler) Set(_ string) {}

func openCentralConnection() (*grpcUtil.LazyClientConn, error) {
	// Create central client.
	centralEndpoint := env.CentralEndpoint.Setting()
	centralClient, err := centralclient.NewClient(centralEndpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "initializing Central client for endpoint %s", env.CentralEndpoint.Setting())
	}

	centralConnFactory := centralclient.NewCentralConnectionFactory(centralClient)
	centralConnection := grpcUtil.NewLazyClientConn()
	certLoader := centralclient.RemoteCertLoader(centralClient)
	go centralConnFactory.SetCentralConnectionWithRetries(&dummyClusterIDHandler{}, centralConnection, certLoader)

	log.Infof("Connecting to Central server %s", centralEndpoint)

	// Wait for connection to be ready.
	okSig := centralConnFactory.OkSignal()
	errSig := centralConnFactory.StopSignal()
	select {
	case <-errSig.Done():
		return nil, errors.Wrap(errSig.Err(), "waiting for Central connection from factory")
	case <-okSig.Done():
	}

	log.Info("Connection setup for gRPC connection to Central finished.")
	return centralConnection, nil
}

// centralHandshake performs the hello-handshake with Central and returns Central's CentralHello reponse on success.
func centralHandshake(ctx context.Context, k8sClient kubernetes.Interface, centralConnection *grpcUtil.LazyClientConn) (*central.CentralHello, error) {
	sensorHello, err := prepareSensorHelloMessage(ctx, k8sClient)
	if err != nil {
		return nil, errors.Wrap(err, "preparing SensorHello message")
	}

	ctx = metadata.AppendToOutgoingContext(ctx, centralsensor.SensorHelloMetadataKey, "true")
	ctx, err = centralsensor.AppendSensorHelloInfoToOutgoingMetadata(ctx, sensorHello)
	if err != nil {
		return nil, errors.Wrap(err, "appending SensorHello info to outgoing metadata")
	}

	client := central.NewSensorServiceClient(centralConnection)
	stream, err := sensorCommon.CommunicateWithAutoSensedEncoding(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "creating bidirectional stream with auto-sensed encoding")
	}

	rawHdr, err := stream.Header()
	if err != nil {
		return nil, errors.Wrap(err, "receiving headers from Central")
	}

	hdr := metautils.MD(rawHdr)
	if hdr.Get(centralsensor.SensorHelloMetadataKey) != "true" {
		log.Error("Central did not send the SensorHello metadata key after connection attempt using a cluster registration secret.")
		log.Error("Possible reason: central does not support CRS-based cluster registration.")
		return nil, errors.New("central headers are missing the SensorHello metadata key ")
	}

	err = stream.Send(&central.MsgFromSensor{Msg: &central.MsgFromSensor_Hello{Hello: sensorHello}})
	if err != nil {
		return nil, errors.Wrap(err, "sending SensorHello message to Central")
	}
	log.Debug("Sent SensorHello to Central.")

	firstMsg, err := stream.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "receiving first message from Central")
	}
	log.Debug("Received Central response.")

	// That's it, we don't need the central connection any longer for the CRS flow.
	if closeErr := stream.CloseSend(); closeErr != nil {
		log.Warnf("error while trying to close the bidirectional streaming connection to Central: %v.", err)
	}

	centralHello := firstMsg.GetHello()
	if centralHello == nil {
		return nil, errors.Errorf("first message received from central was not CentralHello but of type %T", firstMsg.GetMsg())
	}

	log.Infof("Received CentralHello message from Central %s for Cluster %s.", centralHello.GetCentralId(), centralHello.GetClusterId())

	centralCaps := set.NewFrozenStringSet(centralHello.GetCapabilities()...)
	if !centralCaps.Contains(centralsensor.ClusterRegistrationSecretSupported) {
		return nil, errors.Errorf("this version of central (central ID: %s) does not support CRS-based cluster registration", centralHello.GetCentralId())
	}

	return centralHello, nil
}

// persistCertificates persists as Kubernetes Secrets the certificates and keys retrieved from Central during the cluster-registration handshake.
func persistCertificates(ctx context.Context, certsFileMap map[string]string, k8sClient kubernetes.Interface) error {
	for fileName := range certsFileMap {
		log.Debugf("Received certificate from Central named %s.", fileName)
	}

	podName := os.Getenv("POD_NAME")
	sensorNamespace := pods.GetPodNamespace()
	log.Infof("Persisting retrieved certificates as Kubernetes Secrets in namespace %q.", sensorNamespace)
	secretsClient := k8sClient.CoreV1().Secrets(sensorNamespace)

	typedServiceCerts, unknownServices, err := protoconv.ConvertFileMapToTypedServiceCertificateSet(certsFileMap)
	if err != nil {
		return errors.Wrap(err, "converting file map into typed service certificate set")
	}
	if len(unknownServices) > 0 {
		for idx, svc := range unknownServices {
			unknownServices[idx] = fmt.Sprintf("%q", svc)
		}
		unknownServicesJoined := strings.Join(unknownServices, ", ")
		log.Warnf("Central's certificate bundle contained certificates for the following unknown services: %s.", unknownServicesJoined)
	}
	ownerRef, err := certrefresh.FetchSensorDeploymentOwnerRef(ctx, podName, sensorNamespace, k8sClient, wait.Backoff{})
	if err != nil {
		return errors.Wrap(err, "fetching sensor deployment owner reference")
	}
	repository := securedcluster.NewServiceCertificatesRepo(*ownerRef, sensorNamespace, secretsClient)
	persistedCertificates, err := repository.EnsureServiceCertificates(ctx, typedServiceCerts)
	if err != nil {
		return errors.Wrap(err, "ensuring service certificates are persisted")
	}

	serviceTypeNames := getServiceTypeNames(persistedCertificates)
	log.Infof("Successfully persisted received certificates for: %v.", strings.Join(serviceTypeNames, ", "))

	return nil
}

func getServiceTypeNames(serviceCertificates []*storage.TypedServiceCertificate) []string {
	serviceTypeNames := make([]string, 0, len(serviceCertificates))
	for _, c := range serviceCertificates {
		serviceTypeNames = append(serviceTypeNames, c.ServiceType.String())
	}
	return serviceTypeNames
}

// prepareSensorHelloMessage assembles the SensorHello message to be sent to Central for the hello-handshake.
func prepareSensorHelloMessage(ctx context.Context, k8sClient kubernetes.Interface) (*central.SensorHello, error) {
	deploymentIdentification := sensor.FetchDeploymentIdentification(ctx, k8sClient)
	log.Infof("Sensor deployment identification for this secured cluster: %s.", protoutils.NewWrapper(deploymentIdentification))
	helmManagedConfigInit, err := helm.GetHelmManagedConfig(storage.ServiceType_REGISTRANT_SERVICE)
	if err != nil {
		return nil, errors.Wrap(err, "assembling Helm configuration")
	}

	return &central.SensorHello{
		SensorVersion:            version.GetMainVersion(),
		PolicyVersion:            policyversion.CurrentVersion().String(),
		DeploymentIdentification: deploymentIdentification,
		HelmManagedConfigInit:    helmManagedConfigInit,
	}, nil
}
