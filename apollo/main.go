package main

import (
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/db/inmem"
	"bitbucket.org/stack-rox/apollo/apollo/grpc"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/apollo/listeners"
	_ "bitbucket.org/stack-rox/apollo/apollo/listeners/all"
	"bitbucket.org/stack-rox/apollo/apollo/service"
	"bitbucket.org/stack-rox/apollo/apollo/types"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("apollo/main")
)

var imageProcessor *imageprocessor.ImageProcessor
var database db.Storage

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	lis := "docker"
	listenerCreator, exists := listeners.Registry[lis]
	if !exists {
		log.Fatalf("Listener %v does not exist", lis)
	}
	listener, err := listenerCreator()
	if err != nil {
		panic(err)
	}

	database = inmem.New()
	imageProcessor, err = imageprocessor.New(database)
	if err != nil {
		panic(err)
	}

	ruleService := service.NewRuleService(database, imageProcessor)
	grpc.Register(ruleService)

	benchmarkService := service.NewBenchmarkService(database)
	grpc.Register(benchmarkService)

	registryService := service.NewRegistryService(database, imageProcessor)
	grpc.Register(registryService)

	scannerService := service.NewScannerService(database, imageProcessor)
	grpc.Register(scannerService)

	// Initialize by getting resources initially
	runningContainers, err := listener.GetContainers()
	if err != nil {
		panic(err)
	}

	for _, container := range runningContainers {
		alert, err := imageProcessor.Process(container.Image)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Infof("Alert %+v generated for %v", alert, container.ID)
	}

	go listener.Start()

	apiImp := grpc.NewAPI()
	apiImp.Start()

	for {
		select {
		case event := <-listener.Events():
			switch event.Action {
			case types.Create, types.Update:
				for _, container := range event.Containers {
					alert, err := imageProcessor.Process(container.Image)
					if err != nil {
						log.Error(err)
						continue
					}
					log.Infof("Alert %+v generated for %v", alert, container.ID)
				}
			default:
				log.Infof("Event Action %v is currently not implemented", event.Action.String())
			}
		case sig := <-sigs:
			log.Infof("Caught %s signal", sig)
			listener.Done()
			return
		}
	}

	// docker service create --restart-policy=none --name crawler1 -e url=http://blog.alexellis.io -d crawl_site alexellis2/href-counter

}
