package crs

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/env"
	grpcUtil "github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/certdistribution"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const crsTempDirEnvVarName = "ROX_CRS_TMP_DIR"

var log = logging.LoggerForModule()

// EnsureClusterRegistered initiates the CRS based cluster registration flow in case a
// CRS is found instead of regular service certificate.
func EnsureClusterRegistered() error {
	crsTmpDir := os.Getenv(crsTempDirEnvVarName)
	if crsTmpDir == "" {
		log.Errorf("environment variable %s must point to a directory suitable for writing sensitive data to", crsTempDirEnvVarName)
		os.Exit(1)
	}

	log.Infof("Ensuring Secured Cluster is registered.")
	clientconn.SetUserAgent(fmt.Sprintf("%s CSR", clientconn.Sensor))

	// Check if we service certificates are missing.
	_, err := mtls.LeafCertificateFromFile()
	if err == nil {
		// Standard certificates already exist.
		log.Infof("Service certificates found.")
		return nil
	}
	if !os.IsNotExist(err) {
		log.Errorf("Failed to check for service certificate existence: %v", err)
		return errors.Wrap(err, "failure while retrieving service certificates")
	}

	// Service certificates not found.
	log.Infof("Service certificates not found, trying to retrieve cluster registration secret (CRS)")
	crs, err := crs.Load()
	if err != nil {
		log.Errorf("failed to load CRS: %v", err)
		return errors.Wrap(err, "loading CRS")
	}

	// Extract CA certificate.
	var caCert string
	if len(crs.CAs) > 0 {
		caCert = crs.CAs[0]
	}
	if caCert == "" {
		return errors.New("empty CA in CRS")
	}

	// Extract registrator client certificate.
	clientCert, err := tls.X509KeyPair([]byte(crs.Cert), []byte(crs.Key))
	if err != nil {
		return errors.Wrap(err, "parsing CRS certificate")
	}

	// Store certificates and key in crs-tmp volume.
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

	// Now centralConnection is usable.

	sensorHello := &central.SensorHello{
		SensorVersion: version.GetMainVersion(),
		PolicyVersion: policyversion.CurrentVersion().String(),
		// DeploymentIdentification: configHandler.GetDeploymentIdentification(),
		// SensorState:              s.getSensorState(),
		// RequestDeduperState:      s.clientReconcile,
	}

	// Inject desired Helm configuration, if any.
	// TODO(mclasmei): inject actual Helm config
	// if helmManagedCfg := configHandler.GetHelmManagedConfig(); helmManagedCfg != nil && helmManagedCfg.GetClusterId() == "" {
	// 	cachedClusterID, err := helmconfig.LoadCachedClusterID()
	// 	if err != nil {
	// 		log.Warnf("Failed to load cached cluster ID: %s", err)
	// 	} else if cachedClusterID != "" {
	// 		helmManagedCfg = helmManagedCfg.CloneVT()
	// 		helmManagedCfg.ClusterId = cachedClusterID
	// 		log.Infof("Re-using cluster ID %s of previous run. If you see the connection to central failing, re-apply a new Helm configuration via 'helm upgrade', or delete the sensor pod.", cachedClusterID)
	// 	}

	// 	sensorHello.HelmManagedConfigInit = helmManagedCfg
	// }

	// Prepare outgoing context.
	ctx := context.Background()

	ctx = metadata.AppendToOutgoingContext(ctx, centralsensor.SensorHelloMetadataKey, "true")
	ctx, err = centralsensor.AppendSensorHelloInfoToOutgoingMetadata(ctx, sensorHello)
	if err != nil {
		return errors.Wrap(err, "appending SensorHello to outoing metadata")
	}

	client := central.NewSensorServiceClient(centralConnection)

	stream, err := communicateWithAutoSensedEncoding(ctx, client)
	if err != nil {
		return err
	}

	rawHdr, err := stream.Header()
	if err != nil {
		return errors.Wrap(err, "receiving headers from central")
	}
	hdr := metautils.MD(rawHdr)
	if hdr.Get(centralsensor.SensorHelloMetadataKey) != "true" {
		return errors.New("central headers is missing SensorHello metadata key")
	}

	err = stream.Send(&central.MsgFromSensor{Msg: &central.MsgFromSensor_Hello{Hello: sensorHello}})
	if err != nil {
		return errors.Wrap(err, "sending SensorHello message to Central")
	}
	log.Info("Sent SensorHello to Central")

	firstMsg, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving first message from central")
	}
	log.Info("Received Central response")

	centralHello := firstMsg.GetHello()
	if centralHello == nil {
		return errors.Errorf("first message received from central was not CentralHello but of type %T", firstMsg.GetMsg())
	}
	log.Info("Received CentralHello")

	clusterID := centralHello.GetClusterId()
	log.Infof("ClusterID = %s", clusterID)
	log.Infof("CentralID = %s", centralHello.GetCentralId())

	for k, v := range centralHello.GetCertBundle() {
		fmt.Printf("Certificate: %s\n", k)
		fmt.Println(v)
	}

	err = certdistribution.PersistCertificates(centralHello.GetCertBundle())
	if err != nil {
		return errors.Wrap(err, "persisting certificates")
	}
	log.Infof("Persisted certificates")

	return nil
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
