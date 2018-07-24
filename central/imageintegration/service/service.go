package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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

	GetImageIntegration(ctx context.Context, request *v1.ResourceByID) (*v1.ImageIntegration, error)
	GetImageIntegrations(ctx context.Context, request *v1.GetImageIntegrationsRequest) (*v1.GetImageIntegrationsResponse, error)
	PutImageIntegration(ctx context.Context, request *v1.ImageIntegration) (*empty.Empty, error)
	PostImageIntegration(ctx context.Context, request *v1.ImageIntegration) (*v1.ImageIntegration, error)
	TestImageIntegration(ctx context.Context, request *v1.ImageIntegration) (*empty.Empty, error)
	DeleteImageIntegration(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, detection detection.Detector) Service {
	return &serviceImpl{
		datastore: datastore,
		detector:  detection,
	}
}
