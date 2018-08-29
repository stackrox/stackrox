package service

import (
	"context"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.NetworkPolicyServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(store store.Store, graphEvaluator networkgraph.Evaluator, clusterStore datastore.DataStore) Service {
	return &serviceImpl{
		networkPolicies: store,
		clusterStore:    clusterStore,
		graphEvaluator:  graphEvaluator,
	}
}
