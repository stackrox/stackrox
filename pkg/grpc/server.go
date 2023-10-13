package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/NYTimes/gziphandler"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/audit"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	grpc_errors "github.com/stackrox/rox/pkg/grpc/errors"
	grpc_logging "github.com/stackrox/rox/pkg/grpc/logging"
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/netutil/pipeconn"
	promhttp "github.com/travelaudience/go-promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	// All our gRPC servers should support gzip
	_ "google.golang.org/grpc/encoding/gzip"
)

const (
	defaultMaxMsgSize               = 12 * 1024 * 1024
	defaultMaxResponseMsgSize       = 256 * 1024 * 1024 // 256MB
	defaultMaxGrpcConcurrentStreams = 100               // HTTP/2 spec recommendation for minimum value
)

func init() {
	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_logging.InitGrpcLogger()
}

var (
	log = logging.LoggerForModule()

	// MaxMsgSizeSetting is the setting used for gRPC servers and clients to set maximum receive sizes.
	MaxMsgSizeSetting               = env.RegisterIntegerSetting("ROX_GRPC_MAX_MESSAGE_SIZE", defaultMaxMsgSize)
	maxResponseMsgSizeSetting       = env.RegisterIntegerSetting("ROX_GRPC_MAX_RESPONSE_SIZE", defaultMaxResponseMsgSize)
	maxGrpcConcurrentStreamsSetting = env.RegisterIntegerSetting("ROX_GRPC_MAX_CONCURRENT_STREAMS", defaultMaxGrpcConcurrentStreams)
	enableRequestTracing            = env.RegisterBooleanSetting("ROX_GRPC_ENABLE_REQUEST_TRACING", false)
)

func maxResponseMsgSize() int {
	return maxResponseMsgSizeSetting.IntegerSetting()
}

func maxGrpcConcurrentStreams() uint32 {
	if maxGrpcConcurrentStreamsSetting.IntegerSetting() <= 0 {
		return defaultMaxGrpcConcurrentStreams
	}

	return uint32(maxGrpcConcurrentStreamsSetting.IntegerSetting())
}

type server interface {
	Serve(l net.Listener) error
}

type serverAndListener struct {
	srv      server
	listener net.Listener
	endpoint *EndpointConfig

	stopper func()
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
	Start() *concurrency.ErrorSignal
	// Register adds a new APIService to the list of API services
	Register(services ...APIService)

	// Stop will shutdown all listeners and stop the HTTP/gRPC multiplexed server. This gracefully stops the gRPC
	// server and blocks until all the pending RPCs are finished. Stop returns true if the shutdown process was started.
	// If the server has already stopped, or if another shutdown is in progress, Stop returns false.
	// **Caution:** this should not be called in production unless the application is being shutdown (e.g. termination
	// signal received).
	Stop() bool
}

type apiImpl struct {
	apiServices        []APIService
	config             Config
	requestInfoHandler *requestinfo.Handler
	listeners          []serverAndListener

	grpcServer         *grpc.Server
	shutdownInProgress *atomic.Bool
}

// A Config configures the server.
type Config struct {
	CustomRoutes       []routes.CustomRoute
	IdentityExtractors []authn.IdentityExtractor
	AuthProviders      authproviders.Registry
	Auditor            audit.Auditor

	PreAuthContextEnrichers  []contextutil.ContextUpdater
	PostAuthContextEnrichers []contextutil.ContextUpdater

	// These interceptors are executed post authn and authz.
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
	HTTPInterceptors   []httputil.HTTPInterceptor

	Endpoints []*EndpointConfig

	GRPCMetrics metrics.GRPCMetrics
	HTTPMetrics metrics.HTTPMetrics
}

// NewAPI returns an API object.
func NewAPI(config Config) API {
	var shutdownRequested atomic.Bool
	shutdownRequested.Store(false)
	return &apiImpl{
		config:             config,
		requestInfoHandler: requestinfo.NewRequestInfoHandler(),
		shutdownInProgress: &shutdownRequested,
	}
}

func (a *apiImpl) Start() *concurrency.ErrorSignal {
	startedSig := concurrency.NewErrorSignal()
	if a.shutdownInProgress.Load() {
		startedSig.SignalWithError(errors.New("cannot start gRPC API after Stop was called"))
	} else {
		go a.run(&startedSig)
	}
	return &startedSig
}

func (a *apiImpl) Register(services ...APIService) {
	a.apiServices = append(a.apiServices, services...)
}

func (a *apiImpl) Stop() bool {
	if !a.shutdownInProgress.CompareAndSwap(false, true) {
		return false
	}

	log.Info("Starting stop procedure")
	a.grpcServer.GracefulStop()
	log.Info("gRPC server fully stopped")
	for _, listener := range a.listeners {
		if listener.stopper != nil {
			listener.stopper()
		}
	}
	return true
}

func (a *apiImpl) unaryInterceptors() []grpc.UnaryServerInterceptor {
	// The metrics and error interceptors are first in line, i.e., outermost, to
	// make sure all requests are registered in Prometheus with errors converted
	// to gRPC status codes.
	u := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
		grpc_errors.ErrorToGrpcCodeInterceptor,
		contextutil.UnaryServerInterceptor(a.requestInfoHandler.UpdateContextForGRPC),
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
	if a.config.GRPCMetrics != nil {
		u = append(u, a.config.GRPCMetrics.UnaryMonitoringInterceptor)
	}

	if enableRequestTracing.BooleanSetting() {
		u = append(u, grpc_logging.UnaryServerInterceptor(log))
	}
	return u
}

func (a *apiImpl) streamInterceptors() []grpc.StreamServerInterceptor {
	// The metrics and error interceptors are first in line, i.e., outermost, to
	// make sure all requests are registered in Prometheus with errors converted
	// to gRPC status codes.
	s := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,
		grpc_errors.ErrorToGrpcCodeStreamInterceptor,
		contextutil.StreamServerInterceptor(a.requestInfoHandler.UpdateContextForGRPC),
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
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatal(err)
		}

		if !a.shutdownInProgress.Load() {
			log.Fatal("Unexpected local API server termination.")
		}
	}()
	return dialContext
}

func (a *apiImpl) connectToLocalEndpoint(dialCtxFunc pipeconn.DialContextFunc) (*grpc.ClientConn, error) {
	return grpc.Dial("", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, endpoint string) (net.Conn, error) {
			return dialCtxFunc(ctx)
		}),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxResponseMsgSize())),
		grpc.WithUserAgent(clientconn.GetUserAgent()))
}

func allowPrettyQueryParameter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// checking Values as map[string][]string also catches ?pretty and ?pretty=
		// r.URL.Query().Get("pretty") would not.
		if _, ok := r.URL.Query()["pretty"]; ok {
			r.Header.Set("Accept", "application/json+pretty")
		}
		h.ServeHTTP(w, r)
	})
}

// allowCookiesHeaderMatcher is a header matcher that allows cookies to be returned to the client through the gRPC
// gateway.
func allowCookiesHeaderMatcher(key string) (string, bool) {
	if strings.ToLower(key) == "set-cookie" {
		return "Set-Cookie", true
	}
	return fmt.Sprintf("%s%s", runtime.MetadataHeaderPrefix, key), true
}

func (a *apiImpl) muxer(localConn *grpc.ClientConn) http.Handler {
	contextUpdaters := []contextutil.ContextUpdater{authn.ContextUpdater(a.config.IdentityExtractors...)}
	contextUpdaters = append(contextUpdaters, a.config.PreAuthContextEnrichers...)

	// Interceptors for HTTP/1.1 requests (in order of processing):
	// - RequestInfo handler (consumed by other handlers)
	// - IdentityExtractor
	// - AuthConfigChecker
	preAuthHTTPInterceptors := httputil.ChainInterceptors(
		a.requestInfoHandler.HTTPIntercept,
		contextutil.HTTPInterceptor(contextUpdaters...),
	)

	// Interceptors for HTTP/1.1 requests that must be called after
	// authorization (in order of processing):
	// - Post auth context enrichers, including SAC (with authz tracing if on)
	// - Any other specified interceptors (with authz tracing sink if on)
	postAuthHTTPInterceptors := httputil.ChainInterceptors(
		append([]httputil.HTTPInterceptor{
			contextutil.HTTPInterceptor(a.config.PostAuthContextEnrichers...)},
			a.config.HTTPInterceptors...)...,
	)

	mux := &promhttp.ServeMux{ServeMux: &http.ServeMux{}}
	allRoutes := a.config.CustomRoutes
	for _, apiService := range a.apiServices {
		srvWithRoutes, _ := apiService.(APIServiceWithCustomRoutes)
		if srvWithRoutes == nil {
			continue
		}
		allRoutes = append(allRoutes, srvWithRoutes.CustomRoutes()...)
	}
	for _, route := range allRoutes {
		handler := preAuthHTTPInterceptors(route.Handler(postAuthHTTPInterceptors))

		if a.config.Auditor != nil && route.EnableAudit {
			postAuthHTTPInterceptorsWithAudit := httputil.ChainInterceptors(
				append([]httputil.HTTPInterceptor{a.config.Auditor.PostAuthHTTPInterceptor}, postAuthHTTPInterceptors)...,
			)
			handler = preAuthHTTPInterceptors(route.Handler(postAuthHTTPInterceptorsWithAudit))
		}

		if a.config.HTTPMetrics != nil {
			handler = a.config.HTTPMetrics.WrapHandler(handler, route.Route)
		}
		mux.Handle(route.Route, handler)
	}

	if a.config.AuthProviders != nil {
		mux.Handle(a.config.AuthProviders.URLPathPrefix(), preAuthHTTPInterceptors(a.config.AuthProviders))
	}

	gwMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{EmitDefaults: true}),
		runtime.WithMetadata(a.requestInfoHandler.AnnotateMD),
		runtime.WithOutgoingHeaderMatcher(allowCookiesHeaderMatcher),
		runtime.WithMarshalerOption(
			"application/json+pretty",
			&runtime.JSONPb{
				Indent:       "  ",
				EmitDefaults: true,
			},
		),
	)
	if localConn != nil {
		for _, service := range a.apiServices {
			if err := service.RegisterServiceHandler(context.Background(), gwMux, localConn); err != nil {
				log.Panicf("failed to register API service: %v", err)
			}
		}
	}
	mux.Handle("/v1/", allowPrettyQueryParameter(gziphandler.GzipHandler(gwMux)))
	if features.VulnReportingEnhancements.Enabled() {
		mux.Handle("/v2/", allowPrettyQueryParameter(gziphandler.GzipHandler(gwMux)))
	}
	if err := prometheus.Register(mux); err != nil {
		log.Warnf("failed to register Prometheus collector: %v", err)
	}
	return mux
}

func (a *apiImpl) run(startedSig *concurrency.ErrorSignal) {
	if len(a.config.Endpoints) == 0 {
		panic(errors.New("server has no endpoints"))
	}

	a.grpcServer = grpc.NewServer(
		grpc.Creds(credsFromConn{}),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(a.streamInterceptors()...),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(a.unaryInterceptors()...),
		),
		grpc.MaxRecvMsgSize(MaxMsgSizeSetting.IntegerSetting()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: 40 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxConcurrentStreams(maxGrpcConcurrentStreams()),
	)

	for _, service := range a.apiServices {
		service.RegisterServiceServer(a.grpcServer)
	}

	dialCtxFunc := a.listenOnLocalEndpoint(a.grpcServer)
	localConn, err := a.connectToLocalEndpoint(dialCtxFunc)
	if err != nil {
		log.Panicf("Could not connect to local endpoint: %v", err)
	}

	httpHandler := a.muxer(localConn)

	var allSrvAndLiss []serverAndListener
	for _, endpointCfg := range a.config.Endpoints {
		addr, srvAndLiss, err := endpointCfg.instantiate(httpHandler, a.grpcServer)
		if err != nil {
			if endpointCfg.Optional {
				log.Errorf("Failed to instantiate endpoint config of kind %s: %v", endpointCfg.Kind(), err)
			} else {
				log.Panicf("Failed to instantiate endpoint config of kind %s: %v", endpointCfg.Kind(), err)
			}
		} else {
			log.Infof("%s server listening on %s", endpointCfg.Kind(), addr.String())
			allSrvAndLiss = append(allSrvAndLiss, srvAndLiss...)
		}
	}

	errC := make(chan error, len(allSrvAndLiss))

	for _, srvAndLis := range allSrvAndLiss {
		go a.serveBlocking(srvAndLis, errC)
	}

	a.listeners = allSrvAndLiss

	if startedSig != nil {
		startedSig.Signal()
	}

	if err := <-errC; err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (a *apiImpl) serveBlocking(srvAndLis serverAndListener, errC chan<- error) {
	if err := srvAndLis.srv.Serve(srvAndLis.listener); err != nil {
		// If gRPC shutdown was requested, then we should only log that the endpoint is stopping. Otherwise, this is happening
		// for unknown reasons and an error will be reported.
		if a.shutdownInProgress.Load() {
			log.Infof("gRPC Stop requested: Endpoint shutting down: %s %s", srvAndLis.endpoint.Kind(), srvAndLis.listener.Addr())
		} else {
			if srvAndLis.endpoint.Optional {
				log.Errorf("Error serving optional endpoint %s on %s: %v", srvAndLis.endpoint.Kind(), srvAndLis.listener.Addr(), err)
			} else {
				errC <- errors.Wrapf(err, "error serving required endpoint %s on %s: %v", srvAndLis.endpoint.Kind(), srvAndLis.listener.Addr(), err)
			}
		}
	}
}
