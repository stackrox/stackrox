package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	multiplierStore "bitbucket.org/stack-rox/apollo/central/multiplier/store"
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

	GetDeployment(ctx context.Context, request *v1.ResourceByID) (*v1.Deployment, error)
	ListDeployments(ctx context.Context, request *v1.RawQuery) (*v1.ListDeploymentsResponse, error)
	GetLabels(context.Context, *empty.Empty) (*v1.DeploymentLabelsResponse, error)

	GetMultipliers(ctx context.Context, request *empty.Empty) (*v1.GetMultipliersResponse, error)
	AddMultiplier(ctx context.Context, request *v1.Multiplier) (*v1.Multiplier, error)
	UpdateMultiplier(ctx context.Context, request *v1.Multiplier) (*empty.Empty, error)
	RemoveMultiplier(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, multipliers multiplierStore.Store, enricher enrichment.Enricher) Service {
	return &serviceImpl{
		datastore:   datastore,
		multipliers: multipliers,
		enricher:    enricher,
	}
}
