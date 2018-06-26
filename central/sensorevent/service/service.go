package service

import (
	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	deploymentEventStore "bitbucket.org/stack-rox/apollo/central/deploymentevent/store"
	"bitbucket.org/stack-rox/apollo/central/detection"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Service is the GRPC service interface that provides the entry point for processing deployment events.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	RecordEvent(stream v1.SensorEventService_RecordEventServer) error
}

// New returns a new instance of service.
func New(detector *detection.Detector, scorer *risk.Scorer, deploymentEvents deploymentEventStore.Store, images imageDataStore.DataStore, deployments deploymentDataStore.DataStore, clusters clusterDataStore.DataStore) Service {
	return &serviceImpl{
		detector: detector,
		scorer:   scorer,

		deploymentEvents: deploymentEvents,
		images:           images,
		deployments:      deployments,
		clusters:         clusters,
	}
}
