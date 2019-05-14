package service

import (
	"context"

	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService
	v1.NetworkGraphServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service instance using the given DataStore.
func New(store nfDS.ClusterDataStore, deployments dDS.DataStore, graphEvaluator graph.Evaluator) Service {
	return &serviceImpl{
		clusterFlows: store,
		deployments:  deployments,
	}
}
