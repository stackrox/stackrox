package grpc

import (
	"context"
	"crypto/tls"
	golog "log"
	"net"
	"net/http"
	"strings"

	"github.com/NYTimes/gziphandler"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/grpc/authn/mtls"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc"
)

const (
	publicAPIEndpoint     = ":443"
	insecureLocalEndpoint = "localhost:444"
)

func init() {
	grpc_prometheus.EnableHandlingTimeHistogram()
}

var (
	log = logging.LoggerForModule()
)

// APIService is the service interface
type APIService interface {
	RegisterServiceServer(server *grpc.Server)
	RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
}

// API listens for new connections on port 443, and redirects them to the gRPC-Gateway
type API interface {
	// Start runs the API in a goroutine.
	Start()
	// Register adds a new APIService to the list of API services
	Register(services ...APIService)
}

type apiImpl struct {
	apiServices []APIService
	config      Config
}

// A Config configures the server.
type Config struct {
	TLS                verifier.TLSConfigurer
	CustomRoutes       []routes.CustomRoute
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
}

// NewAPI returns an API object.
func NewAPI(config Config) API {
	return &apiImpl{
		config: config,
	}
}

func (a *apiImpl) Start() {
	go a.run()
}

func (a *apiImpl) Register(services ...APIService) {
	a.apiServices = append(a.apiServices, services...)
}

func (a *apiImpl) unaryInterceptors() []grpc.UnaryServerInterceptor {
	u := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
		grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		mtls.UnaryInterceptor(),
	}
	u = append(u, a.config.UnaryInterceptors...)

	// Default to deny all access. This forces services to properly override the AuthFunc.
	u = append(u, grpc_auth.UnaryServerInterceptor(deny.AuthFunc))

	u = append(u, a.unaryRecovery())
	return u
}

func (a *apiImpl) streamInterceptors() []grpc.StreamServerInterceptor {
	s := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,
		grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		mtls.StreamInterceptor(),
	}
	s = append(s, a.config.StreamInterceptors...)

	// Default to deny all access. This forces services to properly override the AuthFunc.
	s = append(s, grpc_auth.StreamServerInterceptor(deny.AuthFunc))

	s = append(s, a.streamRecovery())
	return s
}

func (a *apiImpl) listenOnLocalEndpoint(server *grpc.Server) error {
	lis, err := net.Listen("tcp", insecureLocalEndpoint)
	if err != nil {
		return err
	}

	log.Infof("Launching backend GRPC listener on %v", insecureLocalEndpoint)
	// Launch the GRPC listener
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatal(err)
		}
		log.Fatal("The local API server should never terminate")
	}()
	return nil
}

func (a *apiImpl) connectToLocalEndpoint() (*grpc.ClientConn, error) {
	return grpc.Dial(insecureLocalEndpoint, grpc.WithInsecure())
}

func (a *apiImpl) muxer(localConn *grpc.ClientConn) http.Handler {
	mux := http.NewServeMux()
	for _, route := range a.config.CustomRoutes {
		mux.Handle(route.Route, route.Handler())
	}

	gwMux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{EmitDefaults: true}))
	for _, service := range a.apiServices {
		if err := service.RegisterServiceHandler(context.Background(), gwMux, localConn); err != nil {
			log.Panicf("failed to register API service: %v", err)
		}
	}
	mux.Handle("/v1/", gziphandler.GzipHandler(gwMux))
	return mux
}

func (a *apiImpl) run() {
	tlsConf, err := a.config.TLS.TLSConfig()
	if err != nil {
		panic(err)
	}
	tlsConf.NextProtos = []string{"h2"}

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(a.streamInterceptors()...),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(a.unaryInterceptors()...),
		),
	)

	for _, service := range a.apiServices {
		service.RegisterServiceServer(grpcServer)
	}

	if err := a.listenOnLocalEndpoint(grpcServer); err != nil {
		log.Panicf("Could not listen on local endpoint: %v", err)
	}
	localConn, err := a.connectToLocalEndpoint()
	if err != nil {
		log.Panicf("Could not connect to local endpoint: %v", err)
	}

	listener, err := tls.Listen("tcp", publicAPIEndpoint, tlsConf)
	if err != nil {
		log.Panicf("Could not listen on public API endpoint: %v", err)
	}

	srv := &http.Server{
		Addr:      listener.Addr().String(),
		Handler:   wireOrJSONMuxer(grpcServer, a.muxer(localConn)),
		ErrorLog:  golog.New(httpErrorLogger{}, "", golog.LstdFlags),
		TLSConfig: tlsConf,
	}

	log.Infof("gRPC server started on %s", srv.Addr)
	err = srv.Serve(listener)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return
}

func wireOrJSONMuxer(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}
