package service

import (
	"context"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

	PostCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error)
	PutCluster(ctx context.Context, request *storage.Cluster) (*v1.ClusterResponse, error)
	GetCluster(ctx context.Context, request *v1.ResourceByID) (*v1.ClusterResponse, error)
	GetClusters(ctx context.Context, _ *v1.Empty) (*v1.ClustersList, error)
	DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, riskManager manager.Manager) Service {
	return &serviceImpl{
		datastore:   datastore,
		riskManager: riskManager,
	}
}
