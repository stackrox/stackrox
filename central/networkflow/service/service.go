package service

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	dDS "github.com/stackrox/rox/central/deployment/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	"github.com/stackrox/rox/central/networkflow/datastore/entities"
	"github.com/stackrox/rox/central/sensor/service/connection"
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
func New(store nfDS.ClusterDataStore, entities entities.EntityDataStore, deployments dDS.DataStore, clusters clusterDS.DataStore, sensorConnMgr connection.Manager) Service {
	return newService(store, entities, deployments, clusters, sensorConnMgr)
}

func newService(store nfDS.ClusterDataStore, entities entities.EntityDataStore, deployments dDS.DataStore, clusters clusterDS.DataStore, sensorConnMgr connection.Manager) *serviceImpl {
	return &serviceImpl{
		clusterFlows: store,
		entities:     entities,
		deployments:  deployments,
		clusters:     clusters,

		sensorConnMgr: sensorConnMgr,
	}
}
