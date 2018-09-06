package service

import (
	"context"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/networkpolicies/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
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
func New(store store.Store, deployments deploymentDataStore.DataStore, graphEvaluator networkgraph.Evaluator, clusterStore clusterDataStore.DataStore, notifierStore notifierStore.Store) Service {
	return &serviceImpl{
		deployments:     deployments,
		networkPolicies: store,
		notifierStore:   notifierStore,
		clusterStore:    clusterStore,
		graphEvaluator:  graphEvaluator,
	}
}
