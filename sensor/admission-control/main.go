package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	"github.com/stackrox/rox/sensor/admission-control/service"
	"github.com/stackrox/rox/sensor/admission-control/settingswatch"
)

const (
	webhookEndpoint = ":8443"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	log.Infof("StackRox Sensor Admission Control Service, version %s", version.GetMainVersion())

	utils.Must(mainCmd())
}

func mainCmd() error {
	if !features.AdmissionControlService.Enabled() {
		return errors.New("admission control service is not enabled")
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	// Note that the following call returns immediately (connecting happens in the background), hence this does not
	// delay readiness of the admission-control service even if sensor is unavailable.
	sensorConn, err := clientconn.AuthenticatedGRPCConnection(env.SensorEndpoint.Setting(), mtls.SensorSubject)
	if err != nil {
		log.Errorf("Could not establish a gRPC connection to Sensor: %v. Some features will not work.", err)
	}

	mgr := manager.New(sensorConn)
	if err := mgr.Start(); err != nil {
		return errors.Wrap(err, "starting admission control manager")
	}

	if err := settingswatch.WatchK8sForSettingsUpdatesAsync(mgr.Stopped(), mgr.SettingsUpdateC()); err != nil {
		log.Errorf("Could not watch Kubernetes for settings updates: %v. Functionality might be impacted", err)
	}
	if err := settingswatch.WatchMountPathForSettingsUpdateAsync(mgr.Stopped(), mgr.SettingsUpdateC()); err != nil {
		log.Errorf("Could not watch mount path for settings updates: %v. Functionality might be impacted", err)
	}
	if err := settingswatch.RunSettingsPersister(mgr); err != nil {
		log.Errorf("Could not run settings persister: %v. Admission control service might take longer to become ready after container restarts", err)
	}
	if sensorConn != nil {
		settingswatch.WatchSensorSettingsPush(mgr, sensorConn)
	}

	serverConfig := pkgGRPC.Config{
		Endpoints: []*pkgGRPC.EndpointConfig{
			{
				ListenEndpoint: webhookEndpoint,
				TLS:            verifier.NonCA{},
				ServeHTTP:      true,
			},
		},
	}

	apiServer := pkgGRPC.NewAPI(serverConfig)
	apiServer.Register(service.New(mgr))

	apiServer.Start()

	sig := <-sigC
	log.Infof("Received signal %v", sig)

	return nil
}
