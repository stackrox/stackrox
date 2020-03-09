package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/pkg/debughandler"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/devmode"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/kubernetes/upgrade"
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
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	upgradeCmdHandler, err := upgrade.NewCommandHandler()
	utils.Must(err)

	sharedClientInterface := client.MustCreateInterface()
	s := sensor.CreateSensor(sharedClientInterface, upgradeCmdHandler)
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
