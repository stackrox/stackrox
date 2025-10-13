package service

import (
	"context"

	"github.com/stackrox/rox/central/serviceidentities/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ServiceIdentityServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(dataStore datastore.DataStore) Service {
	return &serviceImpl{
		dataStore: dataStore,
	}
}
