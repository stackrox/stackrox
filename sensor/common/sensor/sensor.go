package sensor

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/benchmarks"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/env"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn/mtls"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/orchestrators"
	sensor "github.com/stackrox/rox/sensor/common"
	networkConnManager "github.com/stackrox/rox/sensor/common/networkflow/manager"
	networkFlowService "github.com/stackrox/rox/sensor/common/networkflow/service"
	signalService "github.com/stackrox/rox/sensor/common/service"
	"google.golang.org/grpc"
)

const (
	retryInterval = 5 * time.Second
)

var (
	customRoutes = []routes.CustomRoute{
		{
			Route:         "/metrics",
			ServerHandler: promhttp.Handler(),
			Authorizer:    allow.Anonymous(),
			Compression:   false,
		},
	}
)

// A Sensor object configures a StackRox Sensor.
// Its functions execute common tasks across supported platforms.
type Sensor struct {
	logger *logging.Logger

	clusterID          string
	centralEndpoint    string
	advertisedEndpoint string
	image              string

	listener           listeners.Listener
	enforcer           enforcers.Enforcer
	orchestrator       orchestrators.Orchestrator
	networkConnManager networkConnManager.Manager

	server         pkgGRPC.API
	benchScheduler *benchmarks.SchedulerClient

	conn *grpc.ClientConn

	sensorInstance sensor.Sensor
}

// NewSensor initializes a Sensor, including reading configurations from the environment.
func NewSensor(log *logging.Logger, l listeners.Listener, e enforcers.Enforcer, o orchestrators.Orchestrator, n networkConnManager.Manager) *Sensor {
	return &Sensor{
		logger: log,

		clusterID:          env.ClusterID.Setting(),
		centralEndpoint:    env.CentralEndpoint.Setting(),
		advertisedEndpoint: env.AdvertisedEndpoint.Setting(),
		image:              env.Image.Setting(),

		listener:           l,
		enforcer:           e,
		orchestrator:       o,
		networkConnManager: n,
	}
}

// Start registers APIs and starts background tasks.
// It returns once tasks have succesfully started.
func (s *Sensor) Start() {
	// Start up connections.
	s.logger.Infof("Connecting to Central server %s", s.centralEndpoint)
	var err error
	s.conn, err = clientconn.GRPCConnection(s.centralEndpoint)
	if err != nil {
		s.logger.Fatalf("Error connecting to central: %s", err)
	}

	s.benchScheduler, err = benchmarks.NewSchedulerClient(s.orchestrator, s.advertisedEndpoint, s.image, s.conn, s.clusterID)
	if err != nil {
		panic(err)
	}

	// Create grpc server with custom routes
	config := pkgGRPC.Config{
		TLS:                verifier.NonCA{},
		CustomRoutes:       customRoutes,
		UnaryInterceptors:  []grpc.UnaryServerInterceptor{mtls.UnaryInterceptor()},
		StreamInterceptors: []grpc.StreamServerInterceptor{mtls.StreamInterceptor()},
	}
	s.server = pkgGRPC.NewAPI(config)

	s.registerAPIServices()
	s.server.Start()

	// Start all of our channels and listeners
	if s.listener != nil {
		go s.listener.Start()
	}
	if s.enforcer != nil {
		go s.enforcer.Start()
	}
	if s.benchScheduler != nil {
		go s.benchScheduler.Start()
	}

	if s.networkConnManager != nil {
		go s.networkConnManager.Start()
	}
	// Wait for central so we can initiate our GRPC connection to send sensor events.
	s.waitUntilCentralIsReady(s.conn)

	// If everything is brought up correctly, start the sensor.
	if s.listener != nil && s.enforcer != nil {
		go s.runSensor()
	}

	s.logger.Info("Sensor started")
}

// Stop shuts down background tasks.
func (s *Sensor) Stop() {
	// Stop the sensor.
	s.sensorInstance.Stop(nil)

	// Stop all of our listeners.
	if s.listener != nil {
		s.listener.Stop()
	}
	if s.enforcer != nil {
		s.enforcer.Stop()
	}
	if s.benchScheduler != nil {
		s.benchScheduler.Stop()
	}
	if s.networkConnManager != nil {
		s.networkConnManager.Stop()
	}

	s.logger.Info("Sensor shutdown complete")
}

func (s *Sensor) registerAPIServices() {
	s.server.Register(
		benchmarks.NewBenchmarkResultsService(benchmarks.NewLRURelayer(s.conn)),
		signalService.Singleton(),
		networkFlowService.Singleton(),
	)
	s.logger.Info("API services registered")
}

// Function does not complete until central is pingable.
func (s *Sensor) waitUntilCentralIsReady(conn *grpc.ClientConn) {
	pingService := v1.NewPingServiceClient(conn)
	err := pingWithTimeout(pingService)
	for err != nil {
		s.logger.Infof("Ping to Central failed: %s. Retrying...", err)
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

func (s *Sensor) runSensor() {
	s.sensorInstance = sensor.NewSensor(s.conn, s.clusterID)
	for {
		s.logger.Info("Starting central connection.")
		go s.sensorInstance.Start(s.listener.Events(), signalService.Singleton().Indicators(), s.networkConnManager.FlowUpdates(), s.enforcer.Actions())

		if err := s.sensorInstance.Wait(); err != nil {
			s.logger.Errorf("Central connection encountered error: %v. Sleeping for %v", err, retryInterval)
			time.Sleep(retryInterval)
		} else {
			s.logger.Info("Terminating central connection.")
			return
		}
	}
}
