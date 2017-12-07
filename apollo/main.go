package main

import (
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/db/boltdb"
	"bitbucket.org/stack-rox/apollo/apollo/db/inmem"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	_ "bitbucket.org/stack-rox/apollo/apollo/registries/all"
	_ "bitbucket.org/stack-rox/apollo/apollo/scanners/all"
	"bitbucket.org/stack-rox/apollo/apollo/scheduler"
	"bitbucket.org/stack-rox/apollo/apollo/service"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("main")
)

func main() {
	apollo := newApollo()

	var err error
	persistence, err := boltdb.NewWithDefaults("/var/lib/")
	if err != nil {
		panic(err)
	}
	apollo.database = inmem.New(persistence)
	if err = apollo.database.Load(); err != nil {
		log.Fatal(err)
	}

	apollo.imageProcessor, err = imageprocessor.New(apollo.database)
	if err != nil {
		panic(err)
	}

	apollo.benchScheduler = scheduler.NewDockerBenchScheduler()

	go apollo.startGRPCServer()

	apollo.processForever()
}

type apollo struct {
	signalsC       chan (os.Signal)
	benchScheduler *scheduler.DockerBenchScheduler
	imageProcessor *imageprocessor.ImageProcessor
	database       db.Storage
	server         grpc.API
}

func newApollo() *apollo {
	apollo := &apollo{}

	apollo.signalsC = make(chan os.Signal, 1)
	signal.Notify(apollo.signalsC, os.Interrupt)
	signal.Notify(apollo.signalsC, syscall.SIGINT, syscall.SIGTERM)

	return apollo
}

func (a *apollo) startGRPCServer() {
	a.server = grpc.NewAPI()
	a.server.Register(service.NewAgentEventService(a.imageProcessor, a.database))
	a.server.Register(service.NewAlertService(a.database))
	a.server.Register(service.NewBenchmarkService(a.database, a.benchScheduler))
	a.server.Register(service.NewBenchmarkResultsService(a.database))
	a.server.Register(service.NewImagePolicyService(a.database, a.imageProcessor))
	a.server.Register(service.NewRegistryService(a.database, a.imageProcessor))
	a.server.Register(service.NewScannerService(a.database, a.imageProcessor))
	a.server.Start()
}

func (a *apollo) processForever() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Caught panic in process loop; restarting. Stack: %s", string(debug.Stack()))
			a.processForever()
		}
	}()

	for {
		select {
		case sig := <-a.signalsC:
			log.Infof("Caught %s signal", sig)
			log.Infof("Apollo terminated")
			return
		}
	}
}
