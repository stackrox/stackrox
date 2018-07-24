package service

import (
	"context"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
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

	GetSummaryCounts(context.Context, *empty.Empty) (*v1.SummaryCountsResponse, error)
}

// New returns a new Service instance using the given DataStore.
func New(alerts alertDataStore.DataStore, clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore, images imageDataStore.DataStore) Service {
	s := &serviceImpl{
		alerts:      alerts,
		clusters:    clusters,
		deployments: deployments,
		images:      images,
	}
	s.initializeAuthorizer()
	return s
}
