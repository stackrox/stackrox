package service

import (
	"context"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
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

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetSummaryCounts(context.Context, *v1.Empty) (*v1.SummaryCountsResponse, error)
}

// New returns a new Service instance using the given DataStore.
func New(alerts alertDataStore.DataStore, clusters clusterDataStore.DataStore,
	deployments deploymentDataStore.DataStore, images imageDataStore.DataStore,
	secrets secretDataStore.DataStore, nodes nodeDataStore.DataStore) Service {
	s := &serviceImpl{
		alerts:      alerts,
		clusters:    clusters,
		deployments: deployments,
		images:      images,
		secrets:     secrets,
		nodes:       nodes,
	}
	s.initializeAuthorizer()
	return s
}
