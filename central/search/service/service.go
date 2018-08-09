package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"golang.org/x/net/context"
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

	Search(ctx context.Context, request *v1.RawSearchRequest) (*v1.SearchResponse, error)
	Options(ctx context.Context, request *v1.SearchOptionsRequest) (*v1.SearchOptionsResponse, error)
}

// New returns a new Service instance using the given DataStore.
func New(alerts alertDataStore.DataStore, deployments deploymentDataStore.DataStore, images imageDataStore.DataStore, policies policyDataStore.DataStore) Service {
	s := &serviceImpl{
		alerts:      alerts,
		deployments: deployments,
		images:      images,
		policies:    policies,
		parser:      &search.QueryParser{},
	}
	s.initializeAuthorizer()
	return s
}
