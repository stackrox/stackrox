package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	GetAuthStatus(ctx context.Context, request *empty.Empty) (*v1.AuthStatus, error)
}

// New returns a new Service instance using the given DataStore.
func New() Service {
	return &serviceImpl{}
}
