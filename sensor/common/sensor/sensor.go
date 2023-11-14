package sensor

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	serviceAuthn "github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/routes"
	grpcUtil "github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/kocache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/chaos"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/image"
	"github.com/stackrox/rox/sensor/common/reconciliation"
	"github.com/stackrox/rox/sensor/common/scannerdefinitions"
)

const (
	// The 127.0.0.1 ensures we do not expose it externally and must be port-forwarded to
	pprofServer = "127.0.0.1:6060"

	publicAPIEndpoint = ":8443"

	publicWebhookEndpoint = ":9443"
)

var (
	log = logging.LoggerForModule()
)

// A Sensor object configures a StackRox Sensor.
// Its functions execute common tasks across supported platforms.
type Sensor struct {
	centralEndpoint    string
	advertisedEndpoint string

	configHandler config.Handler
	detector      detector.Detector
	imageService  image.Service
	components    []common.SensorComponent
	apiServices   []pkgGRPC.APIService

	server          pkgGRPC.API
	webhookServer   pkgGRPC.API
	profilingServer *http.Server

	currentState    common.SensorComponentEvent
	currentStateMtx *sync.Mutex

	centralConnection        *grpcUtil.LazyClientConn
	centralCommunication     CentralCommunication
	centralConnectionFactory centralclient.CentralConnectionFactory

	stoppedSig concurrency.ErrorSignal

	notifyList            []common.Notifiable
	reconnect             atomic.Bool
	reconcile             atomic.Bool
	deduperStateProcessor *reconciliation.DeduperStateProcessor
}

// NewSensor initializes a Sensor, including reading configurations from the environment.
func NewSensor(
	configHandler config.Handler,
	detector detector.Detector,
	imageService image.Service,
	centralConnectionFactory centralclient.CentralConnectionFactory,
	deduperStateProcessor *reconciliation.DeduperStateProcessor,
	components ...common.SensorComponent,
) *Sensor {
	return &Sensor{
		centralEndpoint:    env.CentralEndpoint.Setting(),
		advertisedEndpoint: env.AdvertisedEndpoint.Setting(),

		configHandler:         configHandler,
		detector:              detector,
		imageService:          imageService,
		deduperStateProcessor: deduperStateProcessor,
		components:            append(components, detector, configHandler, deduperStateProcessor), // Explicitly add the config handler

		centralConnectionFactory: centralConnectionFactory,
		centralConnection:        grpcUtil.NewLazyClientConn(),

		currentState:    common.SensorComponentEventOfflineMode,
		currentStateMtx: &sync.Mutex{},

		stoppedSig: concurrency.NewErrorSignal(),

		reconnect: atomic.Bool{},
	}
}

// AddAPIServices adds the api services to the sensor. It should be called PRIOR to Start()
func (s *Sensor) AddAPIServices(services ...pkgGRPC.APIService) {
	s.apiServices = append(s.apiServices, services...)
}

// AddNotifiable adds a common.Notifiable component to the list of components that will be notified of any connectivity
// state changes. All components passed to NewSensor are added by default.
func (s *Sensor) AddNotifiable(notifiable common.Notifiable) {
	s.notifyList = append(s.notifyList, notifiable)
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

// offlineAwareProbeSource is an interface that abstracts the functionality of loading a kernel probe.
type offlineAwareProbeSource interface {
	probeupload.ProbeSource
	offlineAware
}

func createKOCacheSource(centralEndpoint string) (offlineAwareProbeSource, error) {
	kernelObjsBaseURL := "/kernel-objects"
	kernelObjsClient, err := clientconn.NewHTTPClient(mtls.CentralSubject, centralEndpoint, 0)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating central HTTP transport")
	}
	return kocache.New(context.Background(), kernelObjsClient, kernelObjsBaseURL, kocache.StartOffline()), nil
}

// Start registers APIs and starts background tasks.
// It returns once tasks have successfully started.
func (s *Sensor) Start() {
	// Start up connections.
	log.Infof("Connecting to Central server %s", s.centralEndpoint)
	if chaos.HasChaosProxy() {
		chaos.InitializeChaosConfiguration(context.Background())
	}

	go s.centralConnectionFactory.SetCentralConnectionWithRetries(s.centralConnection)

	for _, c := range s.components {
		s.AddNotifiable(c)
		switch v := c.(type) {
		case common.CentralGRPCConnAware:
			v.SetCentralGRPCClient(s.centralConnection)
		}
	}
	s.imageService.SetClient(s.centralConnection)
	s.profilingServer = s.startProfilingServer()

	var centralReachable concurrency.Flag

	legacyAdmissionControllerRoute := routes.CustomRoute{
		Route:         "/admissioncontroller",
		Authorizer:    allow.Anonymous(),
		ServerHandler: &readinessHandler{centralReachable: &centralReachable},
		Compression:   false,
	}
	readinessRoute := routes.CustomRoute{
		Route:         "/ready",
		Authorizer:    allow.Anonymous(),
		ServerHandler: &readinessHandler{centralReachable: &centralReachable},
		Compression:   false,
	}

	customRoutes := []routes.CustomRoute{readinessRoute, legacyAdmissionControllerRoute}

	koCacheSource, err := createKOCacheSource(s.centralEndpoint)
	if err != nil {
		utils.Should(errors.Wrap(err, "Failed to create kernel object download/caching layer"))
	} else {
		probeDownloadHandler := probeupload.NewConnectionAwareProbeHandler(probeupload.LogCallback(log), koCacheSource)
		koCacheRoute := routes.CustomRoute{
			Route:         "/kernel-objects/",
			Authorizer:    idcheck.CollectorOnly(),
			ServerHandler: http.StripPrefix("/kernel-objects", probeDownloadHandler),
			Compression:   false, // kernel objects are compressed
		}
		customRoutes = append(customRoutes, koCacheRoute)
		s.AddNotifiable(wrapNotifiable(probeDownloadHandler, "Kernel probe server handler"))
		s.AddNotifiable(wrapNotifiable(koCacheSource, "Kernel object cache"))
	}

	// Enable endpoint to retrieve vulnerability definitions if local image scanning is enabled.
	if env.LocalImageScanningEnabled.BooleanSetting() {
		route, err := s.newScannerDefinitionsRoute(s.centralEndpoint)
		if err != nil {
			utils.Should(errors.Wrap(err, "Failed to create scanner definition route"))
		}
		customRoutes = append(customRoutes, *route)
	}

	// Create grpc server with custom routes
	mtlsServiceIDExtractor, err := serviceAuthn.NewExtractor()
	if err != nil {
		log.Panicf("Error creating mTLS-based service identity extractor: %v", err)
	}

	conf := pkgGRPC.Config{
		CustomRoutes:       customRoutes,
		IdentityExtractors: []authn.IdentityExtractor{mtlsServiceIDExtractor},
		Endpoints: []*pkgGRPC.EndpointConfig{
			{
				ListenEndpoint: publicAPIEndpoint,
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      true,
			},
		},
	}
	s.server = pkgGRPC.NewAPI(conf)

	s.server.Register(s.apiServices...)
	log.Info("API services registered")

	s.server.Start()

	webhookConfig := pkgGRPC.Config{
		CustomRoutes: []routes.CustomRoute{legacyAdmissionControllerRoute, readinessRoute},
		Endpoints: []*pkgGRPC.EndpointConfig{
			{
				ListenEndpoint: publicWebhookEndpoint,
				TLS:            verifier.NonCA{},
				ServeHTTP:      true,
			},
		},
	}

	s.webhookServer = pkgGRPC.NewAPI(webhookConfig)
	s.webhookServer.Start()

	for _, component := range s.components {
		if err := component.Start(); err != nil {
			utils.Should(errors.Wrapf(err, "sensor component %T failed to start", component))
		}
	}
	log.Info("All components have started")

	okSig := s.centralConnectionFactory.OkSignal()
	errSig := s.centralConnectionFactory.StopSignal()

	if features.PreventSensorRestartOnDisconnect.Enabled() {
		log.Info("Running Sensor with connection retry: preventing sensor restart on disconnect")
		go s.communicationWithCentralWithRetries(&centralReachable)
	} else {
		log.Info("Running Sensor without connection retries: sensor will restart on disconnect")
		// This has to be checked only if retries are not enabled. With retries, this signal will be checked
		// inside communicationWithCentralWithRetries since it has to be re-checked on reconnects, and not
		// crash if it fails.
		select {
		case <-errSig.Done():
			s.stoppedSig.SignalWithErrorWrap(errSig.Err(), "getting connection from connection factory")
			return
		case <-okSig.Done():
			s.changeState(common.SensorComponentEventCentralReachable)
		case <-s.stoppedSig.Done():
			return
		}
		go s.communicationWithCentral(&centralReachable)
	}
}

// newScannerDefinitionsRoute returns a custom route that serves scanner
// definitions retrieved from Central.
func (s *Sensor) newScannerDefinitionsRoute(centralEndpoint string) (*routes.CustomRoute, error) {
	handler, err := scannerdefinitions.NewDefinitionsHandler(centralEndpoint)
	if err != nil {
		return nil, err
	}
	s.AddNotifiable(handler)
	// We rely on central to handle content encoding negotiation.
	return &routes.CustomRoute{
		Route:         "/scanner/definitions",
		Authorizer:    idcheck.ScannerOnly(),
		ServerHandler: handler,
	}, nil
}

// Stop shuts down background tasks.
func (s *Sensor) Stop() {
	if features.PreventSensorRestartOnDisconnect.Enabled() {
		s.stoppedSig.Signal()
	} else {
		// Stop communication with central.
		if s.centralConnection != nil {
			s.centralCommunication.Stop(nil)
		}
	}

	for _, c := range s.components {
		c.Stop(nil)
	}

	log.Infof("Sensor stop was called. Stopping all listeners")

	if s.profilingServer != nil {
		if err := s.profilingServer.Close(); err != nil {
			log.Errorf("Error closing profiling server: %v", err)
		}
	}

	if s.server != nil && !s.server.Stop() {
		log.Warnf("Sensor gRPC server stop was called more than once")
	}

	if s.webhookServer != nil && !s.webhookServer.Stop() {
		log.Warnf("Sensor webhook server stop was called more than once")
	}

	log.Info("Sensor shutdown complete")
}

func (s *Sensor) communicationWithCentral(centralReachable *concurrency.Flag) {
	s.centralCommunication = NewCentralCommunication(s.deduperStateProcessor, false, false, s.components...)

	s.centralCommunication.Start(central.NewSensorServiceClient(s.centralConnection), centralReachable, s.configHandler, s.detector)

	if err := s.centralCommunication.Stopped().Wait(); err != nil {
		log.Errorf("Sensor reported an error: %v", err)
		s.stoppedSig.SignalWithError(err)
	} else {
		log.Info("Terminating central connection")
		s.stoppedSig.Signal()
	}
}

func (s *Sensor) changeState(state common.SensorComponentEvent) {
	s.currentStateMtx.Lock()
	defer s.currentStateMtx.Unlock()
	if s.currentState != state {
		log.Infof("Updating Sensor State to: %s", state)
		s.currentState = state
		s.notifyAllComponents(s.currentState)
	}
}

func (s *Sensor) notifyAllComponents(notification common.SensorComponentEvent) {
	for _, component := range s.notifyList {
		component.Notify(notification)
	}
}

func wrapOrNewError(err error, message string) error {
	if err == nil {
		return errors.New(message)
	}
	return errors.Wrap(err, message)
}

func (s *Sensor) communicationWithCentralWithRetries(centralReachable *concurrency.Flag) {
	// Attempt a simple restart strategy: if connection broke, re-establish the connection with exponential back-offs.
	// This approach does not consider messages that were already sent to central_sender but weren't written to the stream.
	// This re-creates the entire gRPC communication stack, and assumes that a reconciliation should be made once the
	// connection is up again.
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = 0 // It never stops if set to 0
	exponential.InitialInterval = env.ConnectionRetryInitialInterval.DurationSetting()
	exponential.MaxInterval = env.ConnectionRetryMaxInterval.DurationSetting()

	if features.SensorReconciliationOnReconnect.Enabled() {
		s.reconcile.Store(true)
	}

	err := backoff.RetryNotify(func() error {
		log.Infof("Attempting connection setup (client reconciliation = %s)", strconv.FormatBool(s.reconcile.Load()))
		select {
		case <-s.centralConnectionFactory.OkSignal().WaitC():
			// Connection is up, we can try to create a new central communication
			s.changeState(common.SensorComponentEventCentralReachable)
		case <-s.centralConnectionFactory.StopSignal().WaitC():
			// Connection is still broken, report and try again
			go s.centralConnectionFactory.SetCentralConnectionWithRetries(s.centralConnection)
			return wrapOrNewError(s.centralConnectionFactory.StopSignal().Err(), "connection couldn't be re-established")
		}

		// At this point, we know that connection factory reported that connection is up.
		// Try to create a central communication component. This component will fail (Stopped() signal) if the connection
		// suddenly broke.
		s.centralCommunication = NewCentralCommunication(s.deduperStateProcessor, s.reconnect.Load(), s.reconcile.Load(), s.components...)
		s.centralCommunication.Start(central.NewSensorServiceClient(s.centralConnection), centralReachable, s.configHandler, s.detector)
		// Reset the exponential back-off if the connection succeeds
		exponential.Reset()
		select {
		case <-s.centralCommunication.Stopped().WaitC():
			if err := s.centralCommunication.Stopped().Err(); err != nil {
				if errors.Is(err, errCantReconcile) {
					if errors.Is(err, errLargePayload) {
						log.Warnf("Deduper payload is too large for sensor to handle. Sensor will reconnect without client reconciliation." +
							"Consider increasing the maximum receive message size in sensor 'ROX_GRPC_MAX_MESSAGE_SIZE'")
					} else {
						log.Warnf("Sensor cannot reconcile due to: %v", err)
					}
					s.reconcile.Store(false)
				}
				log.Infof("Communication with Central stopped with error: %s. Retrying", err)
			} else {
				log.Info("Communication with Central stopped. Retrying")
			}
			// Communication either ended or there was an error. Either way we should retry.
			// Send notification to all components that we are running in offline mode
			s.changeState(common.SensorComponentEventOfflineMode)
			s.centralConnectionFactory.Reset()
			s.reconnect.Store(true)
			// Trigger goroutine that will attempt the connection. s.centralConnectionFactory.*Signal() should be
			// checked to probe connection state.
			go s.centralConnectionFactory.SetCentralConnectionWithRetries(s.centralConnection)
			return wrapOrNewError(s.centralCommunication.Stopped().Err(), "communication stopped")
		case <-s.stoppedSig.WaitC():
			// This means sensor was signaled to finish, this error shouldn't be retried
			log.Info("Received stop signal from Sensor. Stopping without retrying")
			s.centralCommunication.Stop(nil)
			return backoff.Permanent(wrapOrNewError(s.stoppedSig.Err(), "received sensor stop signal"))
		}
	}, exponential, func(err error, d time.Duration) {
		log.Infof("Central communication stopped: %s. Retrying after %s...", err, d.Round(time.Second))
	})

	log.Info("Stopping gRPC connection retry loop")

	if err != nil {
		log.Warnf("Backoff returned error: %s", err)
	}
}

// Stopped returns an error signal that returns when the sensor terminates.
func (s *Sensor) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedSig
}
