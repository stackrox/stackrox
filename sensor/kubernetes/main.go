package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/continuousprofiling"
	"github.com/stackrox/rox/pkg/devmode"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/cloudproviders/gcp"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/crs"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"golang.org/x/sys/unix"
)

var log = logging.LoggerForModule()

func init() {
	memlimit.SetMemoryLimit()
}

func main() {
	premain.StartMain()

	devmode.StartOnDevBuilds("kubernetes")

	if err := continuousprofiling.SetupClient(continuousprofiling.DefaultConfig(),
		continuousprofiling.WithDefaultAppName("sensor")); err != nil {
		log.Errorf("unable to start continuous profiling: %v", err)
	}

	log.Infof("Running StackRox Version: %s", version.GetMainVersion())

	features.LogFeatureFlags()

	if len(os.Args) > 1 && os.Args[1] == "ensure-service-certificates" {
		err := crs.EnsureServiceCertificatesPresent()
		if err != nil {
			log.Errorf("Ensuring presence of service certificates for this cluster failed: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Start the prometheus metrics server
	metrics.NewServer(metrics.SensorSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
	metrics.GatherThrottleMetricsForever(metrics.SensorSubsystem.String())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	var sharedClientInterface client.Interface
	var sharedClientInterfaceForFetchingPodOwnership client.Interface

	// Workload manager is only non-nil when we are mocking out the k8s client
	workloadManager := fake.NewWorkloadManager(fake.ConfigDefaults())
	if workloadManager != nil {
		sharedClientInterface = workloadManager.Client()
		sharedClientInterfaceForFetchingPodOwnership = client.MustCreateInterface()
	} else {
		sharedClientInterface = client.MustCreateInterface()
	}
	clientconn.SetUserAgent(clientconn.Sensor)
	centralClient, err := centralclient.NewClient(env.CentralEndpoint.Setting())
	if err != nil {
		utils.CrashOnError(errors.Wrapf(err, "sensor failed to start while initializing central HTTP client for endpoint %s", env.CentralEndpoint.Setting()))
	}
	centralConnFactory := centralclient.NewCentralConnectionFactory(centralClient)
	certLoader := centralclient.RemoteCertLoader(centralClient)

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(sharedClientInterface).
		WithCentralConnectionFactory(centralConnFactory).
		WithCertLoader(certLoader).
		WithWorkloadManager(workloadManager).
		WithIntrospectionK8sClient(sharedClientInterfaceForFetchingPodOwnership))
	utils.CrashOnError(err)

	s.Start()
	gcp.Singleton().Start()

	for {
		select {
		case sig := <-sigs:
			log.Infof("Caught %s signal", sig)
			s.Stop()
			gcp.Singleton().Stop()
		case <-s.Stopped().Done():
			if err := s.Stopped().Err(); err != nil {
				log.Fatalf("Sensor exited with error: %v", err)
			}
			log.Info("Sensor exited normally")
			return
		}
	}
}
