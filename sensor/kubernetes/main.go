package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/benchmarks"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/sensor"
	signalService "github.com/stackrox/rox/pkg/sensor/service"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/orchestrator"
	grpcLib "google.golang.org/grpc"
)

var (
	logger = logging.LoggerForModule()

	clusterID          string
	centralEndpoint    string
	advertisedEndpoint string
	image              string

	server               grpc.API
	listenerInstance     listeners.Listener
	enforcerInstance     enforcers.Enforcer
	benchScheduler       *benchmarks.SchedulerClient
	orchestratorInstance orchestrators.Orchestrator

	conn *grpcLib.ClientConn

	sensorInstance sensor.Sensor
)

const (
	retryInterval = 5 * time.Second
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	initialize()

	start()

	for {
		select {
		case sig := <-sigs:
			logger.Infof("Caught %s signal", sig)
			stop()
			logger.Info("Swarm Sensor terminated")
			return
		}
	}
}

// Fetch all needed environment information and initialize all needed objects.
func initialize() {
	// Read environment.
	clusterID = env.ClusterID.Setting()
	centralEndpoint = env.CentralEndpoint.Setting()
	advertisedEndpoint = env.AdvertisedEndpoint.Setting()
	image = env.Image.Setting()

	// Start up connections.
	var err error
	conn, err = clientconn.GRPCConnection(centralEndpoint)
	if err != nil {
		logger.Fatalf("Error connecting to central: %s", err)
	}

	listenerInstance = listener.New()

	enforcerInstance, err = enforcer.New()
	if err != nil {
		panic(err)
	}

	orchestratorInstance, err = orchestrator.New()
	if err != nil {
		panic(err)
	}

	benchScheduler, err = benchmarks.NewSchedulerClient(orchestratorInstance, advertisedEndpoint, image, conn, clusterID)
	if err != nil {
		panic(err)
	}

	logger.Info("Kubernetes Sensor Initialized")
}

// Start all necessary side processes then start sensor.
func start() {
	// Create grpc server with custom routes
	config := grpc.Config{
		TLS:          verifier.NonCA{},
		CustomRoutes: customRoutes(),
	}
	server = grpc.NewAPI(config)

	logger.Infof("Connecting to Central server %s", centralEndpoint)
	registerAPIServices(conn)
	server.Start()

	// Start all of our channels and listeners
	if listenerInstance != nil {
		go listenerInstance.Start()
	}
	if enforcerInstance != nil {
		go enforcerInstance.Start()
	}
	if benchScheduler != nil {
		go benchScheduler.Start()
	}

	// Wait for central so we can initiate our GRPC connection to send sensor events.
	waitUntilCentralIsReady(conn)

	// If everything is brought up correctly, start the sensor.
	if listenerInstance != nil && enforcerInstance != nil {
		go runSensor()
	}

	logger.Info("Kubernetes Sensor Started")
}

func runSensor() {
	sensorInstance = sensor.NewSensor(conn, clusterID)
	for {
		logger.Info("Starting central connection.")
		go sensorInstance.Start(listenerInstance.Events(), signalService.Singleton().Indicators(), enforcerInstance.Actions())

		if err := sensorInstance.Wait(); err != nil {
			logger.Errorf("Central connection encountered error: %v. Sleeping for %v", err, retryInterval)
			time.Sleep(retryInterval)
		} else {
			logger.Info("Terminating central connection.")
			return
		}
	}
}

// Stop stops the sensor and all necessary side processes.
func stop() {
	// Stop the sensor.
	sensorInstance.Stop(nil)

	// Stop all of our listeners.
	if listenerInstance != nil {
		listenerInstance.Stop()
	}
	if enforcerInstance != nil {
		enforcerInstance.Stop()
	}
	if benchScheduler != nil {
		benchScheduler.Stop()
	}

	logger.Info("Kubernetes Sensor Stopped")
}

// Helper functions.
////////////////////

// Provides the custom routes to provide.
func customRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:         "/metrics",
			ServerHandler: promhttp.Handler(),
			Authorizer:    allow.Anonymous(),
			Compression:   false,
		},
	}
}

// Registers our connection for benchmarking.
func registerAPIServices(conn *grpcLib.ClientConn) {
	server.Register(
		benchmarks.NewBenchmarkResultsService(benchmarks.NewLRURelayer(conn)),
		signalService.Singleton(),
	)
	logger.Info("API services registered")
}

// Function does not complete until central is pingable.
func waitUntilCentralIsReady(conn *grpcLib.ClientConn) {
	pingService := v1.NewPingServiceClient(conn)
	err := pingWithTimeout(pingService)
	for err != nil {
		logger.Infof("Ping to Central failed: %s. Retrying...", err)
		time.Sleep(2 * time.Second)
		err = pingWithTimeout(pingService)
	}
}

// Ping a service with a timeout of 10 seconds.
func pingWithTimeout(svc v1.PingServiceClient) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = svc.Ping(ctx, &v1.Empty{})
	return
}
