package grpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	golog "log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/grpc/auth"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/mtls/verifier"
	"github.com/NYTimes/gziphandler"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const grpcPort = ":443"
const endpoint = "localhost" + grpcPort

var (
	log = logging.New("api")
)

// APIService is the service interface
type APIService interface {
	RegisterServiceServer(server *grpc.Server)
	RegisterServiceHandlerFromEndpoint(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
}

// API listens for new connections on port 443, and redirects them to the gRPC-Gateway
type API interface {
	// Start runs the API in a goroutine.
	Start()
	// Register adds a new APIService to the list of API services
	Register(service APIService)
}

type apiImpl struct {
	apiServices []APIService
	config      Config
}

// A Config configures the server.
type Config struct {
	TLS          verifier.TLSConfigurer
	CustomRoutes map[string]http.Handler
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

// httpErrorLogger implements io.Writer interface. It is used to control
// error messages coming from http server which can be logged.
type httpErrorLogger struct {
}

// Write suppresses EOF error messages
func (l httpErrorLogger) Write(p []byte) (n int, err error) {
	if !bytes.Contains(p, []byte("EOF")) {
		return os.Stderr.Write(p)
	}
	return len(p), nil
}

func (a *apiImpl) Register(service APIService) {
	a.apiServices = append(a.apiServices, service)
}

func panicHandler(p interface{}) (err error) {
	if r := recover(); r == nil {
		err = fmt.Errorf("%v", p)
		log.Errorf("Caught panic in gRPC call. Stack: %s", string(debug.Stack()))
	}
	return
}

func (a *apiImpl) run() {
	tlsConf, err := a.config.TLS.TLSConfig()
	if err != nil {
		panic(err)
	}

	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(panicHandler),
	}
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConf)),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
				auth.StreamInterceptor(),
				grpc_recovery.StreamServerInterceptor(opts...),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
				auth.UnaryInterceptor(),
				grpc_recovery.UnaryServerInterceptor(opts...),
			),
		),
	)

	for _, service := range a.apiServices {
		service.RegisterServiceServer(grpcServer)
	}

	ctx := context.Background()
	dcreds := credentials.NewTLS(&tls.Config{
		ServerName:         endpoint,
		RootCAs:            tlsConf.RootCAs,
		InsecureSkipVerify: true,
	})
	dopts := []grpc.DialOption{grpc.WithTransportCredentials(dcreds)}

	mux := http.NewServeMux()
	for prefix, handler := range a.config.CustomRoutes {
		mux.Handle(prefix, handler)
	}
	// EmitDefaults allows marshalled structs with the omitempty: false setting to return 0-valued defaults.
	gwMux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{EmitDefaults: true}))
	for _, service := range a.apiServices {
		if err := service.RegisterServiceHandlerFromEndpoint(ctx, gwMux, endpoint, dopts); err != nil {
			panic(err)
		}
	}
	mux.Handle("/v1/", gziphandler.GzipHandler(gwMux))
	conn, err := net.Listen("tcp", grpcPort)
	if err != nil {
		panic(err)
	}
	tlsConf.NextProtos = []string{"h2"}
	srv := &http.Server{
		Addr:      endpoint,
		Handler:   grpcHandlerFunc(grpcServer, mux),
		ErrorLog:  golog.New(httpErrorLogger{}, "", golog.LstdFlags),
		TLSConfig: tlsConf,
	}
	fmt.Printf("grpc on port: %v\n", grpcPort)
	err = srv.Serve(tls.NewListener(conn, srv.TLSConfig))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return
}

func grpcHandlerFunc(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}
