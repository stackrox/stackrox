package grpc

import (
	"context"
	"crypto/tls"
	golog "log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/audit"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/alpn"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	downgradingServer "github.com/stackrox/rox/pkg/grpc/http1downgrade/server"
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil/pipeconn"
	"github.com/stackrox/rox/pkg/tlsutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	maxMsgSize                = 8 * 1024 * 1024
	defaultMaxResponseMsgSize = 256 * 1024 * 1024 // 256MB
)

func init() {
	grpc_prometheus.EnableHandlingTimeHistogram()
}

var (
	log = logging.LoggerForModule()

	maxResponseMsgSizeSetting = env.RegisterSetting("ROX_GRPC_MAX_RESPONSE_SIZE")
)

func maxResponseMsgSize() int {
	if setting := maxResponseMsgSizeSetting.Setting(); setting != "" {
		value, err := strconv.Atoi(setting)
		if err == nil {
			return value
		}
		log.Warnf("Invalid value %q for %s: %v", setting, maxResponseMsgSizeSetting.EnvVar(), err)
	}
	return defaultMaxResponseMsgSize
}

type server interface {
	Serve(l net.Listener) error
}

type endpointServersConfig struct {
	httpHandler http.Handler
	grpcServer  *grpc.Server

	endpoint string
	tlsConf  *tls.Config
}

func (c *endpointServersConfig) Kind() string {
	var sb strings.Builder
	if c.tlsConf == nil {
		sb.WriteString("Plaintext")
	} else {
		sb.WriteString("TLS-enabled")
	}
	sb.WriteRune(' ')
	if c.httpHandler != nil && c.grpcServer != nil {
		sb.WriteString("multiplexed HTTP/gRPC")
	} else if c.httpHandler != nil {
		sb.WriteString("HTTP")
	} else if c.grpcServer != nil {
		sb.WriteString("gRPC")
	} else {
		sb.WriteString("dummy")
	}
	return sb.String()
}

// Instantiate returns `serverAndListener`s for the given servers at the respective endpoint.
func (c *endpointServersConfig) Instantiate() (net.Addr, []serverAndListener, error) {
	lis, err := net.Listen("tcp", c.endpoint)
	if err != nil {
		return nil, nil, err
	}

	var httpLis, grpcLis net.Listener

	var result []serverAndListener

	tlsConf := c.tlsConf
	if tlsConf != nil {
		if c.grpcServer != nil {
			tlsConf = alpn.ApplyPureGRPCALPNConfig(tlsConf)
		}
		lis = tls.NewListener(lis, tlsConf)

		if c.grpcServer != nil && c.httpHandler != nil {
			protoMap := map[string]*net.Listener{
				alpn.PureGRPCALPNString: &grpcLis,
				"":                      &httpLis,
			}
			tlsutils.ALPNDemux(lis, protoMap, tlsutils.ALPNDemuxConfig{})
		}
	}

	// Default to listen on the main listener, HTTP first
	if c.httpHandler != nil && httpLis == nil {
		httpLis = lis
	} else if c.grpcServer != nil && grpcLis == nil {
		grpcLis = lis
	}

	kind := c.Kind()

	if httpLis != nil {
		httpHandler := c.httpHandler
		if c.grpcServer != nil {
			httpHandler = downgradingServer.CreateDowngradingHandler(c.grpcServer, c.httpHandler)
		}

		httpSrv := &http.Server{
			Handler:   httpHandler,
			TLSConfig: tlsConf,
			ErrorLog:  golog.New(httpErrorLogger{}, "", golog.LstdFlags),
		}
		result = append(result, serverAndListener{
			srv:      httpSrv,
			listener: httpLis,
			kind:     kind,
		})
	}
	if grpcLis != nil {
		result = append(result, serverAndListener{
			srv:      c.grpcServer,
			listener: grpcLis,
			kind:     kind,
		})

	}

	return lis.Addr(), result, nil
}

type serverAndListener struct {
	srv      server
	listener net.Listener
	kind     string
}

// APIService is the service interface
type APIService interface {
	RegisterServiceServer(server *grpc.Server)
	RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
}

// APIServiceWithCustomRoutes is the interface for a service that also defines custom routes.
type APIServiceWithCustomRoutes interface {
	APIService

	CustomRoutes() []routes.CustomRoute
}

// API listens for new connections on port 443, and redirects them to the gRPC-Gateway
type API interface {
	// Start runs the API in a goroutine, and returns a signal that can be checked for when the API server is started.
	Start() *concurrency.Signal
	// Register adds a new APIService to the list of API services
	Register(services ...APIService)
}

type apiImpl struct {
	apiServices        []APIService
	config             Config
	requestInfoHandler *requestinfo.Handler
}

// A Config configures the server.
type Config struct {
	TLS                verifier.TLSConfigurer
	CustomRoutes       []routes.CustomRoute
	IdentityExtractors []authn.IdentityExtractor
	AuthProviders      authproviders.Registry
	Auditor            audit.Auditor

	PreAuthContextEnrichers  []contextutil.ContextUpdater
	PostAuthContextEnrichers []contextutil.ContextUpdater

	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor

	PublicEndpoint string

	PlaintextEndpoints EndpointsConfig
	SecureEndpoints    EndpointsConfig

	GRPCMetrics metrics.GRPCMetrics
	HTTPMetrics metrics.HTTPMetrics
}

// NewAPI returns an API object.
func NewAPI(config Config) API {
	return &apiImpl{
		config:             config,
		requestInfoHandler: requestinfo.NewRequestInfoHandler(),
	}
}

func (a *apiImpl) Start() *concurrency.Signal {
	startedSig := concurrency.NewSignal()
	go a.run(&startedSig)
	return &startedSig
}

func (a *apiImpl) Register(services ...APIService) {
	a.apiServices = append(a.apiServices, services...)
}

func (a *apiImpl) unaryInterceptors() []grpc.UnaryServerInterceptor {
	u := []grpc.UnaryServerInterceptor{
		contextutil.UnaryServerInterceptor(a.requestInfoHandler.UpdateContextForGRPC),
		grpc_prometheus.UnaryServerInterceptor,
		contextutil.UnaryServerInterceptor(authn.ContextUpdater(a.config.IdentityExtractors...)),
	}

	if len(a.config.PreAuthContextEnrichers) > 0 {
		u = append(u, contextutil.UnaryServerInterceptor(a.config.PreAuthContextEnrichers...))
	}

	// Check auth and update the context with the error
	u = append(u, interceptor.AuthContextUpdaterInterceptor())

	if a.config.Auditor != nil {
		// Audit the request
		u = append(u, a.config.Auditor.UnaryServerInterceptor())
	}

	// Check if there was an auth failure and return error if so
	u = append(u, interceptor.AuthCheckerInterceptor())

	if len(a.config.PostAuthContextEnrichers) > 0 {
		u = append(u, contextutil.UnaryServerInterceptor(a.config.PostAuthContextEnrichers...))
	}

	u = append(u, a.config.UnaryInterceptors...)
	u = append(u, a.unaryRecovery())
	if features.Telemetry.Enabled() && a.config.GRPCMetrics != nil {
		u = append(u, a.config.GRPCMetrics.UnaryMonitoringInterceptor)
	}
	return u
}

func (a *apiImpl) streamInterceptors() []grpc.StreamServerInterceptor {
	s := []grpc.StreamServerInterceptor{
		contextutil.StreamServerInterceptor(a.requestInfoHandler.UpdateContextForGRPC),
		grpc_prometheus.StreamServerInterceptor,
		contextutil.StreamServerInterceptor(
			authn.ContextUpdater(a.config.IdentityExtractors...)),
	}
	if len(a.config.PreAuthContextEnrichers) > 0 {
		s = append(s, contextutil.StreamServerInterceptor(a.config.PreAuthContextEnrichers...))
	}

	// Default to deny all access. This forces services to properly override the AuthFunc.
	s = append(s, grpc_auth.StreamServerInterceptor(deny.AuthFunc))

	if len(a.config.PostAuthContextEnrichers) > 0 {
		s = append(s, contextutil.StreamServerInterceptor(a.config.PostAuthContextEnrichers...))
	}

	s = append(s, a.config.StreamInterceptors...)

	s = append(s, a.streamRecovery())
	return s
}

func (a *apiImpl) listenOnLocalEndpoint(server *grpc.Server) pipeconn.DialContextFunc {
	lis, dialContext := pipeconn.NewPipeListener()

	log.Info("Launching backend gRPC listener")
	// Launch the GRPC listener
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatal(err)
		}
		log.Fatal("The local API server should never terminate")
	}()
	return dialContext
}

func (a *apiImpl) connectToLocalEndpoint(dialCtxFunc pipeconn.DialContextFunc) (*grpc.ClientConn, error) {
	return grpc.Dial("", grpc.WithInsecure(),
		grpc.WithContextDialer(func(ctx context.Context, endpoint string) (net.Conn, error) {
			return dialCtxFunc(ctx)
		}),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxResponseMsgSize())))
}

func (a *apiImpl) muxer(localConn *grpc.ClientConn) http.Handler {
	contextUpdaters := []contextutil.ContextUpdater{authn.ContextUpdater(a.config.IdentityExtractors...)}
	contextUpdaters = append(contextUpdaters, a.config.PreAuthContextEnrichers...)

	// Interceptors for HTTP/1.1 requests (in order of processing):
	// - RequestInfo handler (consumed by other handlers)
	// - IdentityExtractor
	// - AuthConfigChecker
	httpInterceptors := httputil.ChainInterceptors(
		a.requestInfoHandler.HTTPIntercept,
		contextutil.HTTPInterceptor(contextUpdaters...),
	)

	postAuthHTTPInterceptor := contextutil.HTTPInterceptor(a.config.PostAuthContextEnrichers...)

	mux := http.NewServeMux()
	allRoutes := a.config.CustomRoutes
	for _, apiService := range a.apiServices {
		srvWithRoutes, _ := apiService.(APIServiceWithCustomRoutes)
		if srvWithRoutes == nil {
			continue
		}
		allRoutes = append(allRoutes, srvWithRoutes.CustomRoutes()...)
	}
	for _, route := range allRoutes {
		handler := httpInterceptors(route.Handler(postAuthHTTPInterceptor))
		if a.config.HTTPMetrics != nil {
			handler = a.config.HTTPMetrics.WrapHandler(handler, route.Route)
		}
		mux.Handle(route.Route, handler)
	}

	if a.config.AuthProviders != nil {
		mux.Handle(a.config.AuthProviders.URLPathPrefix(), httpInterceptors(a.config.AuthProviders))
	}

	gwMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{EmitDefaults: true}),
		runtime.WithMetadata(a.requestInfoHandler.AnnotateMD))
	if localConn != nil {
		for _, service := range a.apiServices {
			if err := service.RegisterServiceHandler(context.Background(), gwMux, localConn); err != nil {
				log.Panicf("failed to register API service: %v", err)
			}
		}
	}
	mux.Handle("/v1/", gziphandler.GzipHandler(gwMux))
	return mux
}

func endpointsFromConfig(cfg EndpointsConfig, httpHandler http.Handler, grpcServer *grpc.Server, tlsConf *tls.Config) []endpointServersConfig {
	var endpointSrvCfgs []endpointServersConfig
	for _, endpoint := range cfg.MultiplexedEndpoints {
		endpointSrvCfgs = append(endpointSrvCfgs, endpointServersConfig{
			httpHandler: httpHandler,
			grpcServer:  grpcServer,
			endpoint:    endpoint,
			tlsConf:     tlsConf,
		})
	}

	for _, endpoint := range cfg.HTTPEndpoints {
		endpointSrvCfgs = append(endpointSrvCfgs, endpointServersConfig{
			httpHandler: httpHandler,
			endpoint:    endpoint,
			tlsConf:     tlsConf,
		})
	}

	for _, endpoint := range cfg.GRPCEndpoints {
		endpointSrvCfgs = append(endpointSrvCfgs, endpointServersConfig{
			grpcServer: grpcServer,
			endpoint:   endpoint,
			tlsConf:    tlsConf,
		})
	}
	return endpointSrvCfgs
}

func (a *apiImpl) run(startedSig *concurrency.Signal) {
	tlsConf, err := a.config.TLS.TLSConfig()
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(credsFromConn{}),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(a.streamInterceptors()...),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(a.unaryInterceptors()...),
		),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: 40 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	for _, service := range a.apiServices {
		service.RegisterServiceServer(grpcServer)
	}

	dialCtxFunc := a.listenOnLocalEndpoint(grpcServer)
	localConn, err := a.connectToLocalEndpoint(dialCtxFunc)
	if err != nil {
		log.Panicf("Could not connect to local endpoint: %v", err)
	}

	httpHandler := a.muxer(localConn)

	endpointSrvCfgs := []endpointServersConfig{
		{
			httpHandler: httpHandler,
			grpcServer:  grpcServer,
			endpoint:    a.config.PublicEndpoint,
			tlsConf:     tlsConf,
		},
	}

	secureEndpointSrvCfgs := endpointsFromConfig(a.config.SecureEndpoints, httpHandler, grpcServer, tlsConf)
	endpointSrvCfgs = append(endpointSrvCfgs, secureEndpointSrvCfgs...)

	plaintextEndpointSrvCfgs := endpointsFromConfig(a.config.PlaintextEndpoints, httpHandler, grpcServer, nil)
	endpointSrvCfgs = append(endpointSrvCfgs, plaintextEndpointSrvCfgs...)

	var allSrvAndLiss []serverAndListener
	for _, endpointCfg := range endpointSrvCfgs {
		addr, srvAndLiss, err := endpointCfg.Instantiate()
		if err != nil {
			log.Panicf("Failed to instantiate endpoint config of kind %s: %v", endpointCfg.Kind(), err)
		}
		log.Infof("%s server listening on %s", endpointCfg.Kind(), addr.String())
		allSrvAndLiss = append(allSrvAndLiss, srvAndLiss...)
	}

	errC := make(chan error, len(allSrvAndLiss))

	for _, srvAndLis := range allSrvAndLiss {
		go serveBlocking(srvAndLis, errC)
	}

	if startedSig != nil {
		startedSig.Signal()
	}

	if err := <-errC; err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveBlocking(srvAndLis serverAndListener, errC chan<- error) {
	if err := srvAndLis.srv.Serve(srvAndLis.listener); err != nil {
		errC <- errors.Wrapf(err, "error serving %s", srvAndLis.kind)
	}
}
