package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichanddetect"
	"github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
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
func New(registryFactory registries.Factory,
	scannerFactory scanners.Factory,
	toNotify integration.ToNotify,
	datastore datastore.DataStore,
	clusterDatastore clusterDatastore.DataStore,
	enrichAndDetectLoop enrichanddetect.Loop) Service {
	return &serviceImpl{
		registryFactory:     registryFactory,
		scannerFactory:      scannerFactory,
		toNotify:            toNotify,
		datastore:           datastore,
		clusterDatastore:    clusterDatastore,
		enrichAndDetectLoop: enrichAndDetectLoop,
	}
}
