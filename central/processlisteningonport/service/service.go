package service

import (
	"context"

	datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves process listening on ports data.
type Service interface {
	grpc.APIService

	v1.ListeningEndpointsServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service instance using the given DataStore.
func New(store datastore.DataStore) Service {
	return &serviceImpl{
		dataStore: store,
	}
}
