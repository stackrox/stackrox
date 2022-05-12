package sensor

import (
	"context"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
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
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/image"
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
	profilingServer *http.Server

	centralConnection    *grpcUtil.LazyClientConn
	centralCommunication CentralCommunication
	centralRestClient    *centralclient.Client

	stoppedSig concurrency.ErrorSignal
}

// NewSensor initializes a Sensor, including reading configurations from the environment.
func NewSensor(configHandler config.Handler, detector detector.Detector, imageService image.Service, centralClient *centralclient.Client, components ...common.SensorComponent) *Sensor {
	return &Sensor{
		centralEndpoint:    env.CentralEndpoint.Setting(),
		advertisedEndpoint: env.AdvertisedEndpoint.Setting(),

		configHandler:     configHandler,
		detector:          detector,
		imageService:      imageService,
		components:        append(components, detector, configHandler), // Explicitly add the config handler
		centralRestClient: centralClient,
		centralConnection: grpcUtil.NewLazyClientConn(),

		stoppedSig: concurrency.NewErrorSignal(),
	}
}

// AddAPIServices adds the api services to the sensor. It should be called PRIOR to Start()
func (s *Sensor) AddAPIServices(services ...pkgGRPC.APIService) {
	s.apiServices = append(s.apiServices, services...)
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

func createKOCacheSource(centralEndpoint string) (probeupload.ProbeSource, error) {
	kernelObjsBaseURL := "/kernel-objects"
	kernelObjsClient, err := clientconn.NewHTTPClient(mtls.CentralSubject, centralEndpoint, 0)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating central HTTP transport")
	}
	return kocache.New(context.Background(), kernelObjsClient, kernelObjsBaseURL, kocache.Options{}), nil
}

// Start registers APIs and starts background tasks.
// It returns once tasks have successfully started.
func (s *Sensor) Start() {
	// Start up connections.
	log.Infof("Connecting to Central server %s", s.centralEndpoint)

	centralConnSignal := concurrency.NewSignal()
	go s.gRPCConnectToCentralWithRetries(&centralConnSignal)

	for _, c := range s.components {
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
		probeDownloadHandler := probeupload.NewProbeServerHandler(probeupload.LogCallback(log), koCacheSource)
		koCacheRoute := routes.CustomRoute{
			Route:         "/kernel-objects/",
			Authorizer:    idcheck.CollectorOnly(),
			ServerHandler: http.StripPrefix("/kernel-objects", probeDownloadHandler),
			Compression:   false, // kernel objects are compressed
		}
		customRoutes = append(customRoutes, koCacheRoute)
	}

	// Enable endpoint to retrieve vulnerability definitions if local image scanning is enabled.
	if features.LocalImageScanning.Enabled() && env.LocalImageScanningEnabled.BooleanSetting() {
		route, err := newScannerDefinitionsRoute(s.centralEndpoint)
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

	webhookServer := pkgGRPC.NewAPI(webhookConfig)
	webhookServer.Start()

	for _, component := range s.components {
		if err := component.Start(); err != nil {
			_ = utils.Should(errors.Wrapf(err, "sensor component %T failed to start", component))
		}
	}

	select {
	case <-centralConnSignal.Done():
	case <-s.stoppedSig.Done():
		return
	}
	go s.communicationWithCentral(&centralReachable)
}

// newScannerDefinitionsRoute returns a custom route that serves scanner
// definitions retrieved from Central.
func newScannerDefinitionsRoute(centralEndpoint string) (*routes.CustomRoute, error) {
	handler, err := scannerdefinitions.NewDefinitionsHandler(centralEndpoint)
	if err != nil {
		return nil, err
	}
	// We rely on central to handle content encoding negotiation.
	return &routes.CustomRoute{
		Route:         "/scanner/definitions",
		Authorizer:    idcheck.ScannerOnly(),
		ServerHandler: handler,
	}, nil
}

func (s *Sensor) gRPCConnectToCentralWithRetries(signal *concurrency.Signal) {
	opts := []clientconn.ConnectionOption{clientconn.UseServiceCertToken(true)}

	// waits until central is ready and has a valid license, otherwise it kills sensor by sending a signal
	s.waitUntilCentralIsReady()

	certs := s.getCentralTLSCerts()
	if len(certs) != 0 {
		log.Infof("Add %d central CA certs to gRPC connection", len(certs))
		for _, c := range certs {
			log.Infof("Add central CA cert with CommonName: '%s'", c.Subject.CommonName)
		}
		opts = append(opts, clientconn.AddRootCAs(certs...))
	} else {
		log.Infof("Did not add central CA cert to gRPC connection")
	}

	centralConnection, err := clientconn.AuthenticatedGRPCConnection(s.centralEndpoint, mtls.CentralSubject, opts...)
	if err != nil {
		s.stoppedSig.SignalWithErrorWrap(err, "Error connecting to central")
		return
	}
	s.centralConnection.Set(centralConnection)

	signal.Signal()
}

// getCentralTLSCerts only logs errors because this feature should not break
// sensors start-up.
func (s *Sensor) getCentralTLSCerts() []*x509.Certificate {
	certs, err := s.centralRestClient.GetTLSTrustedCerts(context.Background())
	if err != nil {
		log.Warnf("Error fetching centrals TLS certs: %s", err)
	}
	return certs
}

// Stop shuts down background tasks.
func (s *Sensor) Stop() {
	// Stop communication with central.
	if s.centralConnection != nil {
		s.centralCommunication.Stop(nil)
	}

	for _, c := range s.components {
		c.Stop(nil)
	}

	if s.profilingServer != nil {
		if err := s.profilingServer.Close(); err != nil {
			log.Errorf("Error closing profiling server: %v", err)
		}
	}

	log.Info("Sensor shutdown complete")
}

// waitUntilCentralIsReady blocks until central responds with a valid license status on its metadata API,
// or until the retry budget is exhausted (in which case the sensor is marked as stopped and the program
// will exit).
func (s *Sensor) waitUntilCentralIsReady() {
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = 5 * time.Minute
	exponential.MaxInterval = 32 * time.Second
	err := backoff.RetryNotify(func() error {
		return s.pollMetadata()
	}, exponential, func(err error, d time.Duration) {
		log.Infof("Check Central status failed: %s. Retrying after %s...", err, d.Round(time.Millisecond))
	})
	if err != nil {
		s.stoppedSig.SignalWithErrorWrapf(err, "checking central status failed after %s", exponential.GetElapsedTime())
	}
}

// Ping a service with a timeout of 10 seconds.
func (s *Sensor) pollMetadata() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	md, err := s.centralRestClient.GetMetadata(ctx)

	if err != nil {
		return err
	}
	if md.GetLicenseStatus() != v1.Metadata_VALID {
		return errors.Errorf("central license status is not VALID but %v", md.GetLicenseStatus())
	}
	return nil
}

func (s *Sensor) communicationWithCentral(centralReachable *concurrency.Flag) {
	s.centralCommunication = NewCentralCommunication(s.components...)

	s.centralCommunication.Start(s.centralConnection, centralReachable, s.configHandler, s.detector)

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
