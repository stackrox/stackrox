package service

import (
	"context"

	alertDataStore "github.com/stackrox/stackrox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/stackrox/central/image/datastore"
	nodeDataStore "github.com/stackrox/stackrox/central/node/globaldatastore"
	secretDataStore "github.com/stackrox/stackrox/central/secret/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
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
	secrets secretDataStore.DataStore, nodes nodeDataStore.GlobalDataStore) Service {
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
