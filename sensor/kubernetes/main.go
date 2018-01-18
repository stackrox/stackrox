package main

import (
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/stack-rox/apollo/pkg/registries"
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/all"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/all"
	"bitbucket.org/stack-rox/apollo/pkg/sensor"
	"bitbucket.org/stack-rox/apollo/sensor/kubernetes/enforcer"
	"bitbucket.org/stack-rox/apollo/sensor/kubernetes/listener"
	"bitbucket.org/stack-rox/apollo/sensor/kubernetes/orchestrator"
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
	a := sensor.New()

	a.Listener = listener.New()
	a.Enforcer, err = enforcer.New()
	if err != nil {
		a.Logger.Fatal(err)
	}
	a.Orchestrator, err = orchestrator.New()
	if err != nil {
		a.Logger.Fatal(err)
	}
	a.ScannerPoller = scanners.NewScannersClient(a.ApolloEndpoint, a.ClusterID)
	a.RegistryPoller = registries.NewRegistriesClient(a.ApolloEndpoint, a.ClusterID)

	a.Logger.Info("Kubernetes Sensor Initialized")
	return a
}
