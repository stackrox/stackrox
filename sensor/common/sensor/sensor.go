package sensor

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/env"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	serviceAuthn "github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/clusterstatus"
	"github.com/stackrox/rox/sensor/common/compliance"
	networkConnManager "github.com/stackrox/rox/sensor/common/networkflow/manager"
	networkFlowService "github.com/stackrox/rox/sensor/common/networkflow/service"
	"github.com/stackrox/rox/sensor/common/networkpolicies"
	"github.com/stackrox/rox/sensor/common/roxmetadata"
	signalService "github.com/stackrox/rox/sensor/common/signal"
	"google.golang.org/grpc"
)

const (
	// The 127.0.0.1 ensures we do not expose it externally and must be port-forwarded to
	pprofServer = "127.0.0.1:6060"

	publicAPIEndpoint = ":8443"
	localAPIEndpoint  = "127.0.0.1:8444"

	publicWebhookEndpoint = ":9443"
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

	log = logging.LoggerForModule()
)

// A Sensor object configures a StackRox Sensor.
// Its functions execute common tasks across supported platforms.
type Sensor struct {
	clusterID          string
	centralEndpoint    string
	advertisedEndpoint string

	listener                      listeners.Listener
	enforcer                      enforcers.Enforcer
	orchestrator                  orchestrators.Orchestrator
	networkConnManager            networkConnManager.Manager
	commandHandler                compliance.CommandHandler
	networkPoliciesCommandHandler networkpolicies.CommandHandler
	clusterStatusUpdater          clusterstatus.Updater

	server          pkgGRPC.API
	profilingServer *http.Server

	centralConnection    *grpc.ClientConn
	centralCommunication CentralCommunication

	stoppedSig concurrency.ErrorSignal
}

// NewSensor initializes a Sensor, including reading configurations from the environment.
func NewSensor(l listeners.Listener, e enforcers.Enforcer, o orchestrators.Orchestrator, n networkConnManager.Manager,
	m roxmetadata.Metadata, networkPoliciesCommandHandler networkpolicies.CommandHandler, clusterStatusUpdater clusterstatus.Updater) *Sensor {
	return &Sensor{
		clusterID:          env.ClusterID.Setting(),
		centralEndpoint:    env.CentralEndpoint.Setting(),
		advertisedEndpoint: env.AdvertisedEndpoint.Setting(),

		listener:                      l,
		enforcer:                      e,
		orchestrator:                  o,
		networkConnManager:            n,
		commandHandler:                compliance.NewCommandHandler(o, m),
		networkPoliciesCommandHandler: networkPoliciesCommandHandler,
		clusterStatusUpdater:          clusterStatusUpdater,

		stoppedSig: concurrency.NewErrorSignal(),
	}
}

func (s *Sensor) startProfilingServer() *http.Server {
	handler := http.NewServeMux()
	for path, debugHandler := range routes.DebugRoutes {
		handler.Handle(path, debugHandler)
	}
	srv := &http.Server{Addr: pprofServer, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Warnf("Closing profiling server: %v", err)
		}
	}()
	return srv
}

type startable interface {
	Start()
}

// Start registers APIs and starts background tasks.
// It returns once tasks have succesfully started.
func (s *Sensor) Start() {
	// Start up connections.
	log.Infof("Connecting to Central server %s", s.centralEndpoint)
	var err error
	s.centralConnection, err = clientconn.AuthenticatedGRPCConnection(s.centralEndpoint, clientconn.Central)
	if err != nil {
		log.Fatalf("Error connecting to central: %s", err)
	}

	s.profilingServer = s.startProfilingServer()

	var centralReachable concurrency.Flag

	admissionControllerRoute := routes.CustomRoute{
		Route:         "/admissioncontroller",
		Authorizer:    allow.Anonymous(),
		ServerHandler: admissioncontroller.NewHandler(s.centralConnection, &centralReachable),
		Compression:   false,
	}

	customRoutes = append(customRoutes, admissionControllerRoute)

	// Create grpc server with custom routes
	config := pkgGRPC.Config{
		TLS:                   verifier.NonCA{},
		CustomRoutes:          customRoutes,
		IdentityExtractors:    []authn.IdentityExtractor{serviceAuthn.NewExtractor()},
		PublicEndpoint:        publicAPIEndpoint,
		InsecureLocalEndpoint: localAPIEndpoint,
	}
	s.server = pkgGRPC.NewAPI(config)

	s.registerAPIServices()
	s.server.Start()

	webhookConfig := pkgGRPC.Config{
		TLS:            verifier.NonCA{},
		CustomRoutes:   []routes.CustomRoute{admissionControllerRoute},
		PublicEndpoint: publicWebhookEndpoint,
	}

	webhookServer := pkgGRPC.NewAPI(webhookConfig)
	webhookServer.Start()

	// Start all of our channels and listeners
	if s.listener != nil {
		go s.listener.Start()
	}
	if s.enforcer != nil {
		go s.enforcer.Start()
	}
	if s.networkConnManager != nil {
		go s.networkConnManager.Start()
	}
	if s.commandHandler != nil {
		s.commandHandler.Start(compliance.Singleton().Output())
	}

	for _, toStart := range []startable{s.networkPoliciesCommandHandler, s.clusterStatusUpdater} {
		if toStart != nil {
			toStart.Start()
		}
	}

	// Wait for central so we can initiate our GRPC connection to send sensor events.
	s.waitUntilCentralIsReady(s.centralConnection)

	// If everything is brought up correctly, start the sensor.
	if s.listener != nil && s.enforcer != nil {
		go s.communicationWithCentral(&centralReachable)
	}

	if s.orchestrator != nil {
		if err := s.orchestrator.CleanUp(false); err != nil {
			log.Errorf("Could not clean up deployments by previous sensor instances: %v", err)
		}
	}
}

type stoppable interface {
	Stop()
}

// Stop shuts down background tasks.
func (s *Sensor) Stop() {
	// Stop communication with central.
	if s.centralConnection != nil {
		s.centralCommunication.Stop(nil)
	}

	for _, toStop := range []stoppable{s.listener, s.enforcer, s.networkConnManager,
		s.networkPoliciesCommandHandler, s.clusterStatusUpdater} {
		if toStop != nil {
			toStop.Stop()
		}
	}

	if s.profilingServer != nil {
		if err := s.profilingServer.Close(); err != nil {
			log.Errorf("Error closing profiling server: %v", err)
		}
	}
	if s.commandHandler != nil {
		s.commandHandler.Stop(nil)
	}

	if s.orchestrator != nil {
		if err := s.orchestrator.CleanUp(true); err != nil {
			log.Errorf("Could not clean up this sensor's deployments: %v", err)
		}
	}

	log.Info("Sensor shutdown complete")
}

func (s *Sensor) registerAPIServices() {
	s.server.Register(
		signalService.Singleton(),
		networkFlowService.Singleton(),
		compliance.Singleton(),
	)
	log.Info("API services registered")
}

// waitUntilCentralIsReady blocks until central responds with a valid license status on its metadata API,
// or until the retry budget is exhausted (in which case the sensor is marked as stopped and the program
// will exit).
func (s *Sensor) waitUntilCentralIsReady(conn *grpc.ClientConn) {
	const maxRetries = 15
	metadataService := v1.NewMetadataServiceClient(conn)
	err := retry.WithRetry(func() error {
		return pollMetadataWithTimeout(metadataService)
	},
		retry.Tries(maxRetries),
		retry.OnFailedAttempts(func(err error) {
			log.Infof("Check Central status failed: %s. Retrying...", err)
			time.Sleep(2 * time.Second)
		}))
	if err != nil {
		s.stoppedSig.SignalWithErrorf("checking central status failed after %d retries: %v", maxRetries, err)
	}
}

// Ping a service with a timeout of 10 seconds.
func pollMetadataWithTimeout(svc v1.MetadataServiceClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	md, err := svc.GetMetadata(ctx, &v1.Empty{})
	if err != nil {
		return err
	}
	if md.GetLicenseStatus() != v1.Metadata_VALID {
		return errors.Errorf("central license status is not VALID but %v", md.GetLicenseStatus())
	}
	return nil
}

func (s *Sensor) communicationWithCentral(centralReachable *concurrency.Flag) {
	s.centralCommunication = NewCentralCommunication(s.commandHandler, s.enforcer, s.listener, signalService.Singleton(),
		s.networkConnManager, s.networkPoliciesCommandHandler, s.clusterStatusUpdater)
	s.centralCommunication.Start(s.centralConnection, centralReachable)

	if err := s.centralCommunication.Stopped().Wait(); err != nil {
		log.Errorf("Sensor reported an error: %v", err)
		s.stoppedSig.SignalWithError(err)
	} else {
		log.Info("Terminating central connection.")
		s.stoppedSig.Signal()
	}
}

// Stopped returns an error signal that returns when the sensor terminates.
func (s *Sensor) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedSig
}
