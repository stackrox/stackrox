package main

import (
	"os"
	"os/signal"
	"syscall"

	_ "bitbucket.org/stack-rox/apollo/pkg/registries/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/all"

	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/sensor"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
	"bitbucket.org/stack-rox/apollo/sensor/kubernetes/enforcer"
	"bitbucket.org/stack-rox/apollo/sensor/kubernetes/listener"
	"bitbucket.org/stack-rox/apollo/sensor/kubernetes/orchestrator"
)

var (
	logger = logging.LoggerForModule()
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	a := initializeSensor()

	a.Start()

	for {
		select {
		case sig := <-sigs:
			a.Logger.Infof("Caught %s signal", sig)
			a.Stop()
			a.Logger.Info("Kubernetes Sensor terminated")
			return
		}
	}
}

func initializeSensor() *sensor.Sensor {
	var err error

	centralConn, err := clientconn.GRPCConnection(env.CentralEndpoint.Setting())
	if err != nil {
		logger.Fatalf("Error connecting to central: %s", err)
	}

	s := sensor.New(centralConn)

	s.Listener = listener.New()
	s.Enforcer, err = enforcer.New()
	if err != nil {
		s.Logger.Fatal(err)
	}
	s.Orchestrator, err = orchestrator.New()
	if err != nil {
		s.Logger.Fatal(err)
	}
	s.ImageIntegrationPoller = sources.NewImageIntegrationsClient(centralConn, s.ClusterID)

	s.Orchestrator, err = orchestrator.New()
	if err != nil {
		panic(err)
	}

	s.BenchScheduler, err = benchmarks.NewSchedulerClient(s.Orchestrator, s.AdvertisedEndpoint, s.Image, centralConn, s.ClusterID)
	if err != nil {
		panic(err)
	}

	s.ServiceRegistrationFunc = registerAPIServices

	s.Logger.Info("Kubernetes Sensor Initialized")
	return s
}

func registerAPIServices(a *sensor.Sensor) {
	a.Server.Register(benchmarks.NewBenchmarkResultsService(benchmarks.NewLRURelayer(a.Conn)))
	a.Logger.Info("API services registered")
}
