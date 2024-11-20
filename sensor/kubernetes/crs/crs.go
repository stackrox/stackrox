package crs

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/securedcluster"
	"github.com/stackrox/rox/sensor/kubernetes/helm"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const crsTempDirEnvVarName = "ROX_CRS_TMP_DIR"

var (
	log                                  = logging.LoggerForModule()
	fetchSensorDeploymentOwnerRefBackoff = wait.Backoff{
		Duration: 10 * time.Millisecond,
		Factor:   3,
		Jitter:   0.1,
		Steps:    10,
		Cap:      6 * time.Minute,
	}
)

// EnsureClusterRegistered initiates the CRS based cluster registration flow if
//
//  1. no Kubernetes secret named `sensor-tls` exists -- indicating that service-specific
//     secrets have already been set up using e.g. an init-bundle.
//
//     and
//
//  2. no Kubernetes secret named `tls-sensor` exists -- indicating that service-specific
//     secrets have already been retrieved via the CRS-flow (or a regular follow-up certificate
//     rotation).
func EnsureClusterRegistered() error {
	log.Infof("Ensuring Secured Cluster is registered.")
	clientconn.SetUserAgent(fmt.Sprintf("%s CRS", clientconn.Sensor))

	// Check if legacy sensor service certificate, e.g. created via init-bundle, exists.
	if legacySensorServiceCert := os.Getenv(crs.LegacySensorServiceCertEnvName); legacySensorServiceCert != "" {
		log.Infof("Legacy sensor service certificate available, skipping CRS-based cluster registration.")
		return nil
	}

	// Check if modern sensor service certificate, e.g. created by CRS-based cluster registration or
	// refreshed by periodic cert rotation, exists.
	if sensorServiceCert := os.Getenv(crs.SensorServiceCertEnvName); sensorServiceCert != "" {
		log.Infof("Sensor service certificate available, skipping CRS-based cluster registration.")
		return nil
	}

	log.Infof("No sensor service certificate found, initiating CRS-based cluster registration.")
	err := registerCluster()
	if err != nil {
		return errors.Wrap(err, "registering secured cluster")
	}
	log.Info("CRS-based cluster registration complete.")

	return nil

}

func registerCluster() error {
	ctx := context.Background()

	log.Infof("Trying to load CRS.")
	crs, err := crs.Load()
	if err != nil {
		log.Errorf("failed to load CRS: %v", err)
		return errors.Wrap(err, "loading CRS")
	}

	// Store certificates and key in crs-tmp volume, so that we can reference them using
	// the MTLS environment variables and hook them directly into the existing MTLS authentication.
	err = useRegistrationSecret(crs)
	if err != nil {
		return errors.Wrap(err, "preparing registration secret for MTLS authentication")
	}

	// New Kubernetes client.
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return errors.Wrap(err, "obtaining in-cluster Kubernetes config")
	}
	k8sClient := k8sutil.MustCreateK8sClient(config)

	sensorHello, err := prepareSensorHelloMessage(k8sClient)
	if err != nil {
		return errors.Wrap(err, "preparing SensorHello message")
	}

	centralConnection, err := openCentralConnection(ctx, crs)
	if err != nil {
		return errors.Wrap(err, "opening connection to Central")
	}

	centralHello, err := centralHandshake(ctx, crs, sensorHello, centralConnection)
	if err != nil {
		return errors.Wrap(err, "handshake with central")
	}

	// Persisting freshly retrieved service certificates.
	certsFileMap := centralHello.GetCertBundle()
	for fileName := range certsFileMap {
		log.Infof("Received certificate from Central named %s", fileName)
	}
	err = persistCertificates(ctx, certsFileMap, k8sClient)
	if err != nil {
		return errors.Wrap(err, "persisting certificates")
	}

	return nil
}

func useRegistrationSecret(crs *crs.CRS) error {
	crsTmpDir := os.Getenv(crsTempDirEnvVarName)
	if crsTmpDir == "" {
		log.Errorf("environment variable %s must point to a directory suitable for writing sensitive data to", crsTempDirEnvVarName)
		return errors.Errorf("environment variable %s unset", crsTempDirEnvVarName)
	}

	// Extract (first) CA certificate from the CRS.
	var caCert string
	var err error

	if len(crs.CAs) > 0 {
		caCert = crs.CAs[0]
	}
	if caCert == "" {
		return errors.New("malformed Cluster Registration Secret (missing CA certificate)")
	}

	for fileName, spec := range map[string]struct {
		setting env.Setting
		content string
	}{
		"ca.pem":   {setting: mtls.CAFilePathSetting, content: caCert},
		"cert.pem": {setting: mtls.CertFilePathSetting, content: crs.Cert},
		"key.pem":  {setting: mtls.KeyFilePathSetting, content: crs.Key},
	} {
		filePath := filepath.Join(crsTmpDir, fileName)
		envVar := spec.setting.EnvVar()
		err = os.WriteFile(filePath, []byte(spec.content), 0600)
		if err != nil {
			return errors.Wrapf(err, "writing mTLS material to file %q", filePath)
		}
		err = os.Setenv(envVar, filePath)
		if err != nil {
			return errors.Wrapf(err, "setting environment variable %s", envVar)
		}
		log.Infof("Successfully wrote file %s", filePath)
	}

	return nil
}

func openCentralConnection(ctx context.Context, crs *crs.CRS) (*grpcUtil.LazyClientConn, error) {
	// Extract registrator client certificate.
	clientCert, err := tls.X509KeyPair([]byte(crs.Cert), []byte(crs.Key))
	if err != nil {
		return nil, errors.Wrap(err, "parsing CRS certificate")
	}

	// Create central client.
	centralEndpoint := env.CentralEndpoint.Setting()
	centralClient, err := centralclient.NewClientWithCert(centralEndpoint, &clientCert)
	if err != nil {
		return nil, errors.Wrapf(err, "initializing Central client for endpoint %s", env.CentralEndpoint.Setting())
	}

	centralConnFactory := centralclient.NewCentralConnectionFactory(centralClient)
	centralConnection := grpcUtil.NewLazyClientConn()
	certLoader := centralclient.RemoteCertLoader(centralClient)
	go centralConnFactory.SetCentralConnectionWithRetries(centralConnection, certLoader)

	log.Infof("Connecting to Central server %s", centralEndpoint)

	okSig := centralConnFactory.OkSignal()
	errSig := centralConnFactory.StopSignal()
	select {
	case <-errSig.Done():
		log.Errorf("failed to get a connection from Central connection factory: %v", errSig.Err())
		return nil, errors.Wrap(err, "waiting for Central connection from factory")
	case <-okSig.Done():
		log.Info("Central connection ready")
	}

	return centralConnection, nil
}

// Hello Handshake with Central.
func centralHandshake(ctx context.Context, crs *crs.CRS, sensorHello *central.SensorHello, centralConnection *grpcUtil.LazyClientConn) (*central.CentralHello, error) {
	var err error

	ctx = metadata.AppendToOutgoingContext(ctx, centralsensor.SensorHelloMetadataKey, "true")
	ctx, err = centralsensor.AppendSensorHelloInfoToOutgoingMetadata(ctx, sensorHello)
	if err != nil {
		return nil, errors.Wrap(err, "appending SensorHello info to outgoing metadata")
	}

	client := central.NewSensorServiceClient(centralConnection)
	stream, err := communicateWithAutoSensedEncoding(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "creating central stream with auto-sensed encoding")
	}

	rawHdr, err := stream.Header()
	if err != nil {
		return nil, errors.Wrap(err, "receiving headers from central")
	}

	hdr := metautils.MD(rawHdr)
	if hdr.Get(centralsensor.SensorHelloMetadataKey) != "true" {
		return nil, errors.New("central headers is missing SensorHello metadata key")
	}

	err = stream.Send(&central.MsgFromSensor{Msg: &central.MsgFromSensor_Hello{Hello: sensorHello}})
	if err != nil {
		return nil, errors.Wrap(err, "sending SensorHello message to Central")
	}
	log.Debug("Sent SensorHello to Central")

	firstMsg, err := stream.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "receiving first message from central")
	}
	log.Debug("Received Central response")

	centralHello := firstMsg.GetHello()
	if centralHello == nil {
		return nil, errors.Errorf("first message received from central was not CentralHello but of type %T", firstMsg.GetMsg())
	}

	clusterID := centralHello.GetClusterId()
	log.Infof("Received ClusterID %s", clusterID)
	log.Infof("Received CentralID %s", centralHello.GetCentralId())

	centralCaps := set.NewFrozenStringSet(centralHello.GetCapabilities()...)
	if !centralCaps.Contains(centralsensor.ClusterRegistrationSecretSupported) {
		return nil, errors.New("central does not support CRS-based cluster registration")
	}

	return centralHello, nil
}

func persistCertificates(ctx context.Context, certsFileMap map[string]string, k8sClient kubernetes.Interface) error {
	podName := os.Getenv("POD_NAME")
	sensorNamespace := pods.GetPodNamespace()
	log.Infof("Persisting retrieved certificates as Kubernetes Secrets in namespace %q.", sensorNamespace)
	secretsClient := k8sClient.CoreV1().Secrets(sensorNamespace)

	typedServiceCerts, err := protoconv.ConvertFileMapToTypedServiceCertificateSet(certsFileMap)
	if err != nil {
		return errors.Wrap(err, "converting file map into typed service certificate set")
	}
	ownerRef, err := certrefresh.FetchSensorDeploymentOwnerRef(ctx, podName, sensorNamespace, k8sClient, fetchSensorDeploymentOwnerRefBackoff)
	if err != nil {
		return errors.Wrap(err, "fetching sensor deployment owner reference")
	}
	repository := securedcluster.NewServiceCertificatesRepo(*ownerRef, sensorNamespace, secretsClient)
	persistedCertificates, err := repository.EnsureServiceCertificates(ctx, typedServiceCerts)
	if err != nil {
		return errors.Wrap(err, "ensuring service certificates are persisted")
	}

	for _, persistedCert := range persistedCertificates {
		log.Infof("Persisted certificate and key for service %s", persistedCert.ServiceType.String())
	}
	return nil
}

func prepareSensorHelloMessage(k8sClient kubernetes.Interface) (*central.SensorHello, error) {
	// Prepare Hello message.
	deploymentIdentification := sensor.FetchDeploymentIdentification(context.Background(), k8sClient)
	log.Infof("Determined deployment identification: %s", protoutils.NewWrapper(deploymentIdentification))
	helmManagedConfigInit, err := helm.GetHelmManagedConfig(storage.ServiceType_REGISTRANT_SERVICE)
	if err != nil {
		return nil, errors.Wrap(err, "assembling Helm configuration")
	}

	sensorHello := &central.SensorHello{
		SensorVersion:            version.GetMainVersion(),
		PolicyVersion:            policyversion.CurrentVersion().String(),
		DeploymentIdentification: deploymentIdentification,
	}
	sensorHello.HelmManagedConfigInit = helmManagedConfigInit
	return sensorHello, nil
}

func communicateWithAutoSensedEncoding(ctx context.Context, client central.SensorServiceClient) (central.SensorService_CommunicateClient, error) {
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name)}

	for {
		stream, err := client.Communicate(ctx, opts...)
		if err != nil {
			if isUnimplemented(err) && len(opts) > 0 {
				opts = nil
				continue
			}
			return nil, errors.Wrap(err, "opening stream")
		}

		_, err = stream.Header()
		if err != nil {
			if isUnimplemented(err) && len(opts) > 0 {
				opts = nil
				continue
			}
			return nil, errors.Wrap(err, "receiving initial metadata")
		}

		return stream, nil
	}
}

func isUnimplemented(err error) bool {
	spb, ok := status.FromError(err)
	if spb == nil || !ok {
		return false
	}
	return spb.Code() == codes.Unimplemented
}
