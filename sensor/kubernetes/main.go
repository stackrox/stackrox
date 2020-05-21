package main

import (
	"os"
	"os/signal"

	"github.com/stackrox/rox/pkg/debughandler"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/devmode"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/kubernetes/upgrade"
	"golang.org/x/sys/unix"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	premain.StartMain()

	if devbuild.IsEnabled() {
		debughandler.MustStartServerAsync("")

		devmode.StartBinaryWatchdog("kubernetes-sensor")
	}

	log.Infof("Running StackRox Version: %s", version.GetMainVersion())

	// Start the prometheus metrics server
	metrics.NewDefaultHTTPServer().RunForever()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	upgradeCmdHandler, err := upgrade.NewCommandHandler()
	utils.Must(err)

	var sharedClientInterface client.Interface

	// Workload manager is only used when we are mocking out the k8s client
	var workloadManager *fake.WorkloadManager
	if workload := env.FakeKubernetesWorkload.Setting(); workload != "" {
		workloadManager = fake.NewWorkloadManager(workload)
		sharedClientInterface = workloadManager.Client()
	} else {
		sharedClientInterface = client.MustCreateInterface()
	}
	s := sensor.CreateSensor(sharedClientInterface, workloadManager, upgradeCmdHandler)
	s.Start()

	for {
		select {
		case sig := <-sigs:
			log.Infof("Caught %s signal", sig)
			s.Stop()
		case <-s.Stopped().Done():
			if err := s.Stopped().Err(); err != nil {
				log.Fatalf("Sensor exited with error: %v", err)
			} else {
				log.Info("Sensor exited normally")
			}
			return
		}
	}
}
