package main

import (
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/db/boltdb"
	"bitbucket.org/stack-rox/apollo/apollo/db/inmem"
	"bitbucket.org/stack-rox/apollo/apollo/grpc"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/apollo/listeners"
	_ "bitbucket.org/stack-rox/apollo/apollo/listeners/all"
	listenerTypes "bitbucket.org/stack-rox/apollo/apollo/listeners/types"
	"bitbucket.org/stack-rox/apollo/apollo/orchestrators"
	_ "bitbucket.org/stack-rox/apollo/apollo/orchestrators/all"
	_ "bitbucket.org/stack-rox/apollo/apollo/registries/all"
	_ "bitbucket.org/stack-rox/apollo/apollo/scanners/all"
	"bitbucket.org/stack-rox/apollo/apollo/scheduler"
	"bitbucket.org/stack-rox/apollo/apollo/service"
	"bitbucket.org/stack-rox/apollo/apollo/types"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("apollo/main")
)

func main() {
	apollo := newApollo()

	var err error
	persistence, err := boltdb.MakeBoltDB("/var/lib/")
	if err != nil {
		panic(err)
	}
	apollo.database = inmem.New(persistence)
	if err = apollo.database.Load(); err != nil {
		log.Fatal(err)
	}

	const platform = "swarm"
	listenerCreator, exists := listeners.Registry[platform]
	if !exists {
		log.Fatalf("Listener %v does not exist", platform)
	}

	apollo.listener, err = listenerCreator(apollo.database)
	if err != nil {
		panic(err)
	}

	apollo.imageProcessor, err = imageprocessor.New(apollo.database)
	if err != nil {
		panic(err)
	}

	go apollo.startGRPCServer()
	go apollo.listener.Start()

	orchestrator, err := orchestrators.Registry[platform]()
	if err != nil {
		log.Fatal(err)
	}
	apollo.benchScheduler = scheduler.NewDockerBenchScheduler(orchestrator)

	apollo.processForever()
}

type apollo struct {
	signalsC       chan (os.Signal)
	benchScheduler *scheduler.DockerBenchScheduler
	listener       listenerTypes.Listener
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

	alertService := service.NewAlertService(a.database)
	a.server.Register(alertService)

	benchmarkService := service.NewBenchmarkService(a.database, a.benchScheduler)
	a.server.Register(benchmarkService)

	imagePolicyService := service.NewImagePolicyService(a.database, a.imageProcessor)
	a.server.Register(imagePolicyService)

	registryService := service.NewRegistryService(a.database)
	a.server.Register(registryService)

	scannerService := service.NewScannerService(a.database)
	a.server.Register(scannerService)

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
		case event := <-a.listener.Events():
			log.Infof("Received new Deployment Event: %#v", event)
			switch event.Action {
			case types.Create, types.Update:
				alerts, err := a.imageProcessor.Process(event.Deployment)
				if err != nil {
					log.Error(err)
					continue
				}
				for _, alert := range alerts {
					log.Warnf("Alert Generated: %v with Severity %v due to image policy %v", alert.Id, alert.Severity.String(), alert.GetPolicy().GetName())
					for _, violation := range alert.GetPolicy().GetViolations() {
						log.Warnf("\t %v", violation.Message)
					}
					if err := a.database.AddAlert(alert); err != nil {
						log.Error(err)
					}
				}
			default:
				log.Infof("DeploymentEvent Action %v is currently not implemented", event.Action.String())
			}
		case sig := <-a.signalsC:
			log.Infof("Caught %s signal", sig)
			a.listener.Stop()
			log.Infof("Apollo terminated")
			return
		}
	}
}
