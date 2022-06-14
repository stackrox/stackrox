package service

import (
	"context"

	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/networkbaseline/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves pod data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.NetworkBaselineServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, manager manager.Manager) Service {
	return &serviceImpl{
		datastore: datastore,
		manager:   manager,
	}
}
