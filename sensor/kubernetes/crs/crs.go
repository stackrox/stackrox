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
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/env"
	grpcUtil "github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/pods"
	protoconv "github.com/stackrox/rox/pkg/protoconv/certs"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/sensor/helmconfig"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/securedcluster"
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

// EnsureClusterRegistered initiates the CRS based cluster registration flow in case a
// CRS is found instead of regular service certificate.
func EnsureClusterRegistered() error {
	log.Infof("Ensuring Secured Cluster is registered.")
	clientconn.SetUserAgent(fmt.Sprintf("%s CSR", clientconn.Sensor))

	// Check if we service certificates are missing.
	_, err := mtls.LeafCertificateFromFile()
	if err == nil {
		// Standard certificates already exist.
		log.Infof("Service certificates found, skipping CRS-based cluster registration.")
		return nil
	}
	if !os.IsNotExist(err) {
		log.Errorf("Failed to check for service certificate existence: %v", err)
		return errors.Wrap(err, "checking for existing service certificates")
	}

	log.Infof("Service certificates not found, attempting CRS-based cluster registration.")
	err = registerCluster()
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

	// Extract registrator client certificate.
	clientCert, err := tls.X509KeyPair([]byte(crs.Cert), []byte(crs.Key))
	if err != nil {
		return errors.Wrap(err, "parsing CRS certificate")
	}

	// Store certificates and key in crs-tmp volume, so that we can reference them using
	// the MTLS environment variables and hook them directly into the existing MTLS authentication.
	err = useRegistrationSecret(crs)
	if err != nil {
		return errors.Wrap(err, "preparing registration secret for MTLS authentication")
	}

	// Create central client.
	centralEndpoint := env.CentralEndpoint.Setting()
	centralClient, err := centralclient.NewClientWithCert(centralEndpoint, &clientCert)
	if err != nil {
		return errors.Wrapf(err, "initializing Central client for endpoint %s", env.CentralEndpoint.Setting())
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
		return errors.Wrap(err, "waiting for Central connection from factory")
	case <-okSig.Done():
		log.Info("Central connection ready")
	}

	// New Kubernetes client.
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return errors.Wrap(err, "obtaining in-cluster Kubernetes config")
	}
	k8sClient := k8sutil.MustCreateK8sClient(config)

	// Prepare Hello message.
	deploymentIdentification := sensor.FetchDeploymentIdentification(context.Background(), k8sClient)
	log.Infof("Determined deployment identification: %s", protoutils.NewWrapper(deploymentIdentification))
	helmManagedConfigInit, err := getHelmManagedConfig()
	if err != nil {
		return errors.Wrap(err, "assembling Helm configuration")
	}
	sensorHello := &central.SensorHello{
		SensorVersion:            version.GetMainVersion(),
		PolicyVersion:            policyversion.CurrentVersion().String(),
		DeploymentIdentification: deploymentIdentification,
	}
	sensorHello.HelmManagedConfigInit = helmManagedConfigInit

	// Prepare communication channel towards Central.
	ctx = metadata.AppendToOutgoingContext(ctx, centralsensor.SensorHelloMetadataKey, "true")
	ctx, err = centralsensor.AppendSensorHelloInfoToOutgoingMetadata(ctx, sensorHello)
	if err != nil {
		return errors.Wrap(err, "appending SensorHello to outgoing metadata")
	}
	client := central.NewSensorServiceClient(centralConnection)
	stream, err := communicateWithAutoSensedEncoding(ctx, client)
	if err != nil {
		return errors.Wrap(err, "creating central stream with auto-sensed encoding")
	}

	rawHdr, err := stream.Header()
	if err != nil {
		return errors.Wrap(err, "receiving headers from central")
	}

	hdr := metautils.MD(rawHdr)
	if hdr.Get(centralsensor.SensorHelloMetadataKey) != "true" {
		return errors.New("central headers is missing SensorHello metadata key")
	}

	// Hello Handshake with Central.
	err = stream.Send(&central.MsgFromSensor{Msg: &central.MsgFromSensor_Hello{Hello: sensorHello}})
	if err != nil {
		return errors.Wrap(err, "sending SensorHello message to Central")
	}
	log.Debug("Sent SensorHello to Central")

	firstMsg, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving first message from central")
	}
	log.Debug("Received Central response")

	centralHello := firstMsg.GetHello()
	if centralHello == nil {
		return errors.Errorf("first message received from central was not CentralHello but of type %T", firstMsg.GetMsg())
	}

	clusterID := centralHello.GetClusterId()
	log.Infof("Received ClusterID %s", clusterID)
	log.Infof("Received CentralID %s", centralHello.GetCentralId())

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
		return errors.New("empty CA in CRS")
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
			return errors.Wrapf(err, "writing MTLS material to file %s", filePath)
		}
		err = os.Setenv(envVar, filePath)
		if err != nil {
			return errors.Wrapf(err, "setting environment variable %s", envVar)
		}
		log.Infof("Successfully wrote file %s", filePath)
	}

	return nil
}

func persistCertificates(ctx context.Context, certsFileMap map[string]string, k8sClient kubernetes.Interface) error {
	podName := os.Getenv("POD_NAME")
	sensorNamespace := pods.GetPodNamespace()
	secretsClient := k8sClient.CoreV1().Secrets(sensorNamespace)

	typedServiceCerts := protoconv.ConvertFileMapToTypedServiceCertificateSet(certsFileMap)
	if typedServiceCerts == nil {
		return errors.New("empty typed service certificates")
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

func getHelmManagedConfig() (*central.HelmManagedConfigInit, error) {
	var helmManagedConfig *central.HelmManagedConfigInit
	if configFP := helmconfig.HelmConfigFingerprint.Setting(); configFP != "" {
		var err error
		helmManagedConfig, err = helmconfig.Load()
		if err != nil {
			return nil, errors.Wrap(err, "loading Helm cluster config")
		}
		if helmManagedConfig.GetClusterConfig().GetConfigFingerprint() != configFP {
			return nil, errors.Errorf("fingerprint %q of loaded config does not match expected fingerprint %q, config changes can only be applied via 'helm upgrade' or a similar chart-based mechanism", helmManagedConfig.GetClusterConfig().GetConfigFingerprint(), configFP)
		}
		log.Infof("Loaded Helm cluster configuration with fingerprint %q", configFP)

		if err := helmconfig.CheckEffectiveClusterName(helmManagedConfig); err != nil {
			return nil, errors.Wrap(err, "validating cluster name")
		}
	}

	if helmManagedConfig.GetClusterName() == "" {
		certClusterID, err := clusterid.ParseClusterIDFromServiceCert(storage.ServiceType_REGISTRANT_SERVICE)
		if err != nil {
			return nil, errors.Wrap(err, "parsing cluster ID from service certificate")
		}
		if centralsensor.IsInitCertClusterID(certClusterID) {
			return nil, errors.New("a sensor that uses certificates from an init bundle must have a cluster name specified")
		}
	} else {
		log.Infof("Cluster name from Helm configuration: %q", helmManagedConfig.GetClusterName())
	}
	return helmManagedConfig, nil
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
