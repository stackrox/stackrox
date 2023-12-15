package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/devmode"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"golang.org/x/sys/unix"
)

var (
	log = logging.LoggerForModule()
)

func init() {
	memlimit.SetMemoryLimit()
}

func main() {
	premain.StartMain()

	devmode.StartOnDevBuilds("bin/kubernetes-sensor")

	log.Infof("Running StackRox Version: %s", version.GetMainVersion())

	// Start the prometheus metrics server
	metrics.NewServer(metrics.SensorSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
	metrics.GatherThrottleMetricsForever(metrics.SensorSubsystem.String())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	var sharedClientInterface client.Interface

	// Workload manager is only non-nil when we are mocking out the k8s client
	workloadManager := fake.NewWorkloadManager(fake.ConfigDefaults())
	if workloadManager != nil {
		sharedClientInterface = workloadManager.Client()
	} else {
		sharedClientInterface = client.MustCreateInterface()
	}
	clientconn.SetUserAgent(clientconn.Sensor)
	centralConnFactory, err := centralclient.NewCentralConnectionFactory(env.CentralEndpoint.Setting())
	if err != nil {
		utils.CrashOnError(errors.Wrapf(err, "sensor failed to start while initializing gRPC client to endpoint %s", env.CentralEndpoint.Setting()))
	}

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(sharedClientInterface).
		WithCentralConnectionFactory(centralConnFactory).
		WithWorkloadManager(workloadManager))
	utils.CrashOnError(err)

	s.Start()

	for {
		select {
		case sig := <-sigs:
			log.Infof("Caught %s signal", sig)
			s.Stop()
		case <-s.Stopped().Done():
			if err := s.Stopped().Err(); err != nil {
				log.Fatalf("Sensor exited with error: %v", err)
			}
			log.Info("Sensor exited normally")
			return
		}
	}
}
