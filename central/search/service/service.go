package service

import (
	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	policyDataStore "bitbucket.org/stack-rox/apollo/central/policy/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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
