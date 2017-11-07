package grpc

import (
	"bytes"
	"context"
	golog "log"
	"net"
	"net/http"
	"os"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const grpcEndpoint = "127.0.0.1:8081"
const gateway = "0.0.0.0:8080"

var (
	log = logging.New("api")
)

// APIService is the service interface
type APIService interface {
	RegisterServiceServer(server *grpc.Server)
	RegisterServiceHandlerFromEndpoint(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
}

var apiServices []APIService

// Register adds a new APIService to the list of API services
func Register(service APIService) {
	apiServices = append(apiServices, service)
}

// API listens for new connections on APIPort 8080, and redirects them to the gRPC-Gateway
type API interface {
	Start()
}

type apiImpl struct{}

// NewAPI returns an API object.
func NewAPI() API {
	return &apiImpl{}
}

// Start runs the API in a goroutine.
func (a *apiImpl) Start() {
	go a.run()
}

// authFunc implements grpc_auth.AuthFunc. Our services handle their own authentication, so deny by default otherwise.
func (a *apiImpl) authFunc(ctx context.Context, method string) (context.Context, error) {
	return nil, status.Errorf(codes.Unimplemented, "The URL %v you tried to reach isn't implemented.", method)
}

func (a *apiImpl) run() {
	server := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
				grpc_recovery.StreamServerInterceptor(),
			),
		),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
				grpc_recovery.UnaryServerInterceptor(),
			),
		),
	)

	// EmitDefaults allows marshalled structs with the omitempty: false setting to return 0-valued defaults.
	gwMux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{EmitDefaults: true}))
	for _, service := range apiServices {
		service.RegisterServiceServer(server)
		if err := service.RegisterServiceHandlerFromEndpoint(context.Background(), gwMux, grpcEndpoint, []grpc.DialOption{grpc.WithInsecure()}); err != nil {
			panic(err)
		}
	}

	// Start GRPC listener
	grpcListener, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		panic(err)
	}
	go func() {
		err := server.Serve(grpcListener)
		panic(err)
	}()

	// Start HTTP Server
	httpServer := &http.Server{
		Addr:     grpcEndpoint,
		Handler:  a.defaultHandlerFunc(server, gwMux),
		ErrorLog: golog.New(httpErrorLogger{}, "", golog.LstdFlags),
	}
	listener, err := net.Listen("tcp", gateway)
	if err != nil {
		panic(err)
	}
	log.Infof("API server started on %s", gateway)
	if err := httpServer.Serve(listener); err != nil {
		log.Fatal(err)
		return
	}
}

func (a *apiImpl) defaultHandlerFunc(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
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
