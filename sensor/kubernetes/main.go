package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/pkg/debughandler"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/devmode"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/networkflow/service"
	"github.com/stackrox/rox/sensor/common/sensor"
	signalService "github.com/stackrox/rox/sensor/common/signal"
	"github.com/stackrox/rox/sensor/kubernetes/clusterstatus"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/networkpolicies"
	"github.com/stackrox/rox/sensor/kubernetes/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry"
	k8sUpgrade "github.com/stackrox/rox/sensor/kubernetes/upgrade"
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

	upgradeCmdHandler, err := k8sUpgrade.NewCommandHandler()
	utils.Must(err)

	configHandler := config.NewCommandHandler()

	enforcer := enforcer.MustCreate()
	policyDetector := detector.New(enforcer)
	listener := listener.New(configHandler, policyDetector)

	o := orchestrator.New()
	complianceService := compliance.NewService(o)
	complianceCommandHandler := compliance.NewCommandHandler(complianceService)

	processSignals := signalService.New(policyDetector)

	components := []common.SensorComponent{
		listener,
		enforcer,
		manager.Singleton(),
		networkpolicies.NewCommandHandler(),
		clusterstatus.NewUpdater(),
		upgradeCmdHandler,
		complianceCommandHandler,
		processSignals,
	}

	if features.DiagnosticBundle.Enabled() || features.Telemetry.Enabled() {
		components = append(components, telemetry.NewCommandHandler())
	}

	s := sensor.NewSensor(
		configHandler,
		policyDetector,
		components...,
	)

	s.AddAPIServices(
		service.Singleton(),
		processSignals,
		complianceService,
	)

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
