package service

import (
	"context"

	datastore "github.com/stackrox/rox/central/runtimeconfiguration/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves runtime configuration data
type Service interface {
	grpc.APIService

	v1.CollectorRuntimeConfigurationServiceServer
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service instance using the given DataStore.
func New(store datastore.DataStore, connManager connection.Manager) Service {
	return &serviceImpl{
		dataStore:   store,
		connManager: connManager,
	}
}
