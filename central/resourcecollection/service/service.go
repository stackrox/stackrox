package service

import (
	"context"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.CollectionServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, queryResolver datastore.QueryResolver, deploymentDS deploymentDS.DataStore,
	reportConfigDatastore reportConfigDS.DataStore) Service {
	return &serviceImpl{
		datastore:             datastore,
		queryResolver:         queryResolver,
		deploymentDS:          deploymentDS,
		reportConfigDatastore: reportConfigDatastore,
	}
}
