package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/central/networkgraph"
	"bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
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

	v1.NetworkPolicyServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore store.Store, graphEvaluator networkgraph.Evaluator, clusterStore datastore.DataStore) Service {
	return &serviceImpl{
		store:          datastore,
		clusterStore:   clusterStore,
		graphEvaluator: graphEvaluator,
	}
}
