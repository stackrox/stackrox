package sensor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	serviceAuthn "github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/kocache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
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
	clusterID          string
	centralEndpoint    string
	advertisedEndpoint string

	configHandler config.Handler
	detector      detector.Detector
	components    []common.SensorComponent
	apiServices   []pkgGRPC.APIService

	server          pkgGRPC.API
	profilingServer *http.Server

	centralConnection    *grpc.ClientConn
	centralCommunication CentralCommunication

	stoppedSig concurrency.ErrorSignal
}

// NewSensor initializes a Sensor, including reading configurations from the environment.
func NewSensor(configHandler config.Handler, detector detector.Detector, components ...common.SensorComponent) *Sensor {
	return &Sensor{
		clusterID:          clusterid.Get(),
		centralEndpoint:    env.CentralEndpoint.Setting(),
		advertisedEndpoint: env.AdvertisedEndpoint.Setting(),

		configHandler: configHandler,
		detector:      detector,
		components:    append(components, detector, configHandler), // Explicitly add the config handler

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
	kernelObjsBaseURL := fmt.Sprintf("https://%s/kernel-objects", centralEndpoint)
	serverName, _, _, err := netutil.ParseEndpoint(centralEndpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing central endpoint %q", centralEndpoint)
	}
	if netutil.IsIPAddress(serverName) {
		serverName = mtls.CentralSubject.Hostname()
	}

	tlsConfig, err := clientconn.TLSConfig(mtls.CentralSubject, clientconn.TLSConfigOptions{
		UseClientCert: true,
		ServerName:    serverName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "instantiating TLS config for HTTP access to central")
	}

	tlsConfig.NextProtos = []string{"http/1.1", "http/1.0"}
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		log.Warnf("Could not configure transport for HTTP/2 usage: %v", err)
	}

	kernelObjsClient := &http.Client{
		Transport: transport,
	}

	return kocache.New(context.Background(), kernelObjsClient, kernelObjsBaseURL, kocache.Options{}), nil
}

// Start registers APIs and starts background tasks.
// It returns once tasks have succesfully started.
func (s *Sensor) Start() {
	// Start up connections.
	log.Infof("Connecting to Central server %s", s.centralEndpoint)
	var err error
	s.centralConnection, err = clientconn.AuthenticatedGRPCConnection(s.centralEndpoint, mtls.CentralSubject, clientconn.UseServiceCertToken(true))
	if err != nil {
		log.Fatalf("Error connecting to central: %s", err)
	}
	s.detector.SetClient(s.centralConnection)

	s.profilingServer = s.startProfilingServer()

	var centralReachable concurrency.Flag

	admissionControllerRoute := routes.CustomRoute{
		Route:         "/admissioncontroller",
		Authorizer:    allow.Anonymous(),
		ServerHandler: admissioncontroller.NewHandler(s.centralConnection, &centralReachable, s.configHandler),
		Compression:   false,
	}

	customRoutes := []routes.CustomRoute{admissionControllerRoute}

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

	// Create grpc server with custom routes
	mtlsServiceIDExtractor, err := serviceAuthn.NewExtractor()
	if err != nil {
		log.Panicf("Error creating mTLS-based service identity extractor: %v", err)
	}

	config := pkgGRPC.Config{
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
	s.server = pkgGRPC.NewAPI(config)

	s.server.Register(s.apiServices...)
	log.Info("API services registered")

	s.server.Start()

	webhookConfig := pkgGRPC.Config{
		CustomRoutes: []routes.CustomRoute{admissionControllerRoute},
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
			log.Panicf("Sensor component %T failed to start: %v", component, err)
		}
	}

	// Wait for central so we can initiate our GRPC connection to send sensor events.
	s.waitUntilCentralIsReady(s.centralConnection)

	go s.communicationWithCentral(&centralReachable)
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
