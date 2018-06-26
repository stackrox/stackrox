package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/cluster/datastore"
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

	PostCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error)
	PutCluster(ctx context.Context, request *v1.Cluster) (*v1.ClusterResponse, error)
	GetCluster(ctx context.Context, request *v1.ResourceByID) (*v1.ClusterResponse, error)
	GetClusters(ctx context.Context, _ *empty.Empty) (*v1.ClustersList, error)
	DeleteCluster(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}
