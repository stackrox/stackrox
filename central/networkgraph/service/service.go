package service

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	dDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
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
func New(store nfDS.ClusterDataStore,
	entities networkEntityDS.EntityDataStore,
	networkTreeMgr networktree.Manager,
	deployments dDS.DataStore,
	clusters clusterDS.DataStore,
	networkPolicy networkPolicyDS.DataStore,
	graphConfigDS datastore.DataStore) Service {
	return newService(store, entities, networkTreeMgr, deployments, clusters, networkPolicy, graphConfigDS)
}

func newService(store nfDS.ClusterDataStore,
	entities networkEntityDS.EntityDataStore,
	networkTreeMgr networktree.Manager,
	deployments dDS.DataStore,
	clusters clusterDS.DataStore,
	networkPolicy networkPolicyDS.DataStore,
	graphConfigDS datastore.DataStore) *serviceImpl {
	return &serviceImpl{
		clusterFlows:     store,
		entities:         entities,
		networkTreeMgr:   networkTreeMgr,
		deployments:      deployments,
		clusters:         clusters,
		graphConfig:      graphConfigDS,
		networkPolicy:    networkPolicy,
		clusterSACHelper: sachelper.NewClusterSacHelper(clusters),
	}
}
