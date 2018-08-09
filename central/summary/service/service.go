package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
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
