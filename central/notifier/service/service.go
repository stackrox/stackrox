package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/notifier/store"
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

	GetNotifier(ctx context.Context, request *v1.ResourceByID) (*v1.Notifier, error)
	GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error)
	PutNotifier(ctx context.Context, request *v1.Notifier) (*empty.Empty, error)
	PostNotifier(ctx context.Context, request *v1.Notifier) (*v1.Notifier, error)
	TestNotifier(ctx context.Context, request *v1.Notifier) (*empty.Empty, error)
	DeleteNotifier(ctx context.Context, request *v1.DeleteNotifierRequest) (*empty.Empty, error)
}

type policyDetector interface {
	RemoveNotifier(id string)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store, processor processor.Processor, detector policyDetector) Service {
	return &serviceImpl{
		storage:   storage,
		processor: processor,
		detector:  detector,
	}
}
