package service

import (
	"context"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/central/dnrintegration/store"
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

	GetDNRIntegration(ctx context.Context, req *v1.ResourceByID) (*v1.DNRIntegration, error)
	GetDNRIntegrations(ctx context.Context, req *v1.GetDNRIntegrationsRequest) (*v1.GetDNRIntegrationsResponse, error)
	PostDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*v1.DNRIntegration, error)
	PutDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*v1.DNRIntegration, error)
	DeleteDNRIntegration(ctx context.Context, req *v1.ResourceByID) (*empty.Empty, error)
	TestDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store, clusters clusterDataStore.DataStore) Service {
	return &serviceImpl{
		storage:  storage,
		clusters: clusters,
	}
}
